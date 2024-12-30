package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log"

	"EnergyMonitorAppAPI/pkg/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/coreos/go-oidc"
	"github.com/golang-jwt/jwt"
)

type Response struct {
	Message string `json:"message"`
}
type ClaimsPage struct {
	AccessToken string
	Claims      jwt.MapClaims
}

var (
	provider *oidc.Provider
	//oauth2Config oauth2.Config
)

func init() {
	var err error
	// Initialize OIDC provider
	provider, err = oidc.NewProvider(context.Background(), "https://cognito-idp.us-west-2.amazonaws.com/us-west-2_QMNqg83TF")
	if err != nil {
		log.Fatalf("Failed to create OIDC provider: %v", err)
	}

	// Set up OAuth2 config
	//oauth2Config = oauth2.Config{
	//	ClientID:     config.AppConfig.AWS.Cognito.ClientID,
	//	ClientSecret: config.AppConfig.AWS.Cognito.ClientSecret,
	//	RedirectURL:  config.AppConfig.AWS.Cognito.RedirectURL,
	//	Endpoint:     provider.Endpoint(),
	//	Scopes:       []string{oidc.ScopeOpenID, "email", "openid", "phone"},
	//}
}

func SignUpUser(username, password, email string) error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AppConfig.AWS.Region),
	})
	if err != nil {
		return err
	}

	svc := cognitoidentityprovider.New(sess)

	clientId := config.AppConfig.AWS.Cognito.ClientID
	clientSecret := config.AppConfig.AWS.Cognito.ClientSecret
	secretHash := calculateSecretHash(username, clientId, clientSecret)

	input := &cognitoidentityprovider.SignUpInput{
		ClientId:   aws.String(clientId),
		SecretHash: aws.String(secretHash),
		Username:   aws.String(username),
		Password:   aws.String(password),
		UserAttributes: []*cognitoidentityprovider.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(email),
			},
		},
	}

	_, err = svc.SignUp(input)
	if err != nil {
		return err
	}

	// Auto-confirm the user
	adminConfirmSignUpInput := &cognitoidentityprovider.AdminConfirmSignUpInput{
		UserPoolId: aws.String(config.AppConfig.AWS.Cognito.UserPoolID),
		Username:   aws.String(username),
	}

	_, err = svc.AdminConfirmSignUp(adminConfirmSignUpInput)
	if err != nil {
		return err
	}

	return nil
}

func calculateSecretHash(username, clientId, clientSecret string) string {
	mac := hmac.New(sha256.New, []byte(clientSecret))
	mac.Write([]byte(username + clientId))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func SignInUser(username, password string) (string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AppConfig.AWS.Region),
	})
	if err != nil {
		return "", err
	}

	svc := cognitoidentityprovider.New(sess)

	clientId := config.AppConfig.AWS.Cognito.ClientID
	clientSecret := config.AppConfig.AWS.Cognito.ClientSecret
	secretHash := calculateSecretHash(username, clientId, clientSecret)

	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: aws.String("USER_PASSWORD_AUTH"),
		AuthParameters: map[string]*string{
			"USERNAME":    aws.String(username),
			"PASSWORD":    aws.String(password),
			"SECRET_HASH": aws.String(secretHash),
		},
		ClientId: aws.String(clientId),
	}

	result, err := svc.InitiateAuth(input)

	if err != nil {
		return "", err
	}

	if result.AuthenticationResult == nil {
		return "", errors.New("authentication result is nil")
	}

	return *result.AuthenticationResult.AccessToken, nil
}
