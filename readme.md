# Energy Monitor App API

This is the backend API for the Energy Monitor App, built with Go and deployed on AWS Lambda using API Gateway. The API handles user authentication, energy data inputs, energy history, and alerts.

## API Routes

### Authentication

- **POST /auth/login**
  - Description: Log in a user.
  - Request Body:
    ```json
    {
      "username": "user@example.com",
      "password": "password123"
    }
    ```
  - Response:
    ```json
    {
      "success": true,
      "message": "Login successful",
      "token": "jwt-token"
    }
    ```

- **POST /auth/signup**
  - Description: Sign up a new user.
  - Request Body:
    ```json
    {
      "username": "user@example.com",
      "password": "password123"
    }
    ```
  - Response:
    ```json
    {
      "success": true,
      "message": "Signup successful"
    }
    ```

- **POST /auth/signout**
  - Description: Sign out a user.
  - Request Header: `Authorization: Bearer <jwt-token>`
  - Response:
    ```json
    {
      "message": "Successfully signed out"
    }
    ```

### Energy Inputs

- **POST /energy/input**
  - Description: Save energy data.
  - Request Header: `Authorization: Bearer <jwt-token>`
  - Request Body:
    ```json
    {
      "Energy": 100,
      "Date": "2023-10-01"
    }
    ```
  - Response:
    ```json
    {
      "success": true,
      "message": "Energy data saved"
    }
    ```
	
- **GET /energy/upload?fileName=test-file.txt&fileType=text**
  - Description: Get a presigned URL for uploading energy data.
  - Request Header: `Authorization: Bearer <jwt-token>`
  - Response:
    ```json
    {
      "url": "presigned-url"
    }
    ```


- **POST /energy/upload**
  - Description: Upload energy data to DataBase.
  - Request Header: `Authorization: Bearer <jwt-token>`
  - Request Body: 
	```json
		{
		"s3_url": "presigned-url"
		}
	```
  - Response:
    ```json
    {
      "success": true,
      "message": "Energy data uploaded"
    }
    ```

### Energy History

- **GET /energy/history?startDate=2024-06-01&endDate=2024-06-10**
  - Description: Get energy history as date and energy used.
  - Request Header: `Authorization: Bearer <jwt-token>`
  - Response:
    ```json
    [
      {"50","2024-06-01"},
	  {"60","2024-06-01"},
	  {"70","2024-06-01"}
    ]
    ```

- **GET energy/history?startDate=2024-06-01&endDate=2024-06-10**
  - Description: Get energy summary.
  - Request Header: `Authorization: Bearer <jwt-token>`
  - Response:
    ```json
    {
      "trends":  {
		"totalEnergy":100, 
		"dailyTrends": [{"2024-06-01", "20"},{"2024-06-01", "20"}],
		"weeklyTrends": [{"2024-W51", "20"},{"2024-W39", "20"}],
		"monthlyTrends": [{"2024-06", "20"},{"2024-06", "20"}],
		}
    }
    ```

### Alerts

- **GET /alerts**
  - Description: Get alerts.
  - Response:
    ```json
    {
      "Threshold":30,
	  "UerID":"abc"
    }
    ```

- **POST /alerts**
  - Description: Update alerts.
  - Request Body:
    ```json
    {
      "threshold": 40
    }
    ```
  - Response:
    ```json
    {
      "success": true,
      "message": "Alerts updated"
    }
    ```

## AWS Services Used

- **AWS Lambda**: Serverless compute service to run the backend API.
- **Amazon API Gateway**: Managed service to create, publish, maintain, monitor, and secure the API.
- **Amazon S3**: Object storage service to store and retrieve energy data files.
- **Amazon Cognito**: User authentication and authorization service.
- **Amazon CloudWatch**: Monitoring and logging service to track API and Lambda function performance.
- **Amazon DynamoDB**: Saving and retreiving Energy, threshold data.
- **Amazon SNS**: email user up on reaching threshold.

## Setup and Deployment

### Prerequisites

- Go 1.x
- AWS CLI configured with appropriate permissions
- AWS SAM CLI (optional, for local testing and deployment)

### Build and Deploy

1. **Build the Go application**:
   ```sh
   GOOS=linux GOARCH=amd64 go build -o main main.go