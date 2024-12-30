package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	AWS struct {
		Region  string `yaml:"region"`
		Cognito struct {
			ClientID     string `yaml:"client_id"`
			ClientSecret string `yaml:"client_secret"`
			RedirectURL  string `yaml:"redirect_url"`
			IssuerURL    string `yaml:"issuer_url"`
			UserPoolID   string `yaml:"user_pool_id"`
		} `yaml:"cognito"`
		Lambda struct {
			FunctionName string `yaml:"function_name"`
			Role         string `yaml:"role"`
		} `yaml:"lambda"`
		S3 struct {
			BucketName string `yaml:"bucket_name"`
		} `yaml:"s3"`
	} `yaml:"aws"`
}

var AppConfig Config

func LoadConfig() error {
	file, err := os.Open("../../config.yaml")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&AppConfig)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
}
