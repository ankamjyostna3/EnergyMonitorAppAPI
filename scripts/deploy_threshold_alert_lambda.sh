#!/bin/bash

# Variables

LAMBDA_FUNCTION_NAME="thresholdAlert"
GO_FILE_PATH="../lambdas/thresholdAlert/main.go"
ZIP_FILE="threshold_alert.zip"
ROLE_ARN="arn:aws:iam::985539767168:role/LambdaS3DynamoDBRole"
REGION="us-west-2"
BUILD_DIR="build"

# Create build directory if it doesn't exist
mkdir -p $BUILD_DIR

# Build the Go binary for Linux
echo "Building Go binary..."
GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/main $GO_FILE_PATH

# Check if the build was successful
if [ $? -ne 0 ]; then
    echo "Failed to build Go binary"
    exit 1
fi

# Ensure the bootstrap file exists
if [ ! -f "$BUILD_DIR/bootstrap" ]; then
    echo "Creating bootstrap file..."
    echo '#!/bin/sh' > $BUILD_DIR/bootstrap
    echo 'set -euo pipefail' >> $BUILD_DIR/bootstrap
    echo 'exec /var/task/main' >> $BUILD_DIR/bootstrap
    chmod +x $BUILD_DIR/bootstrap
fi

# Create a deployment package
echo "Creating deployment package..."
cd $BUILD_DIR
zip $ZIP_FILE main bootstrap

# Check if the zip command was successful
if [ $? -ne 0 ]; then
    echo "Failed to create deployment package"
    exit 1
fi
cd ..

# Check if the Lambda function exists
echo "Checking if Lambda function exists..."
aws lambda get-function --function-name $LAMBDA_FUNCTION_NAME --region $REGION

if [ $? -eq 0 ]; then
    # Update the existing Lambda function
    echo "Updating Lambda function..."
    aws lambda update-function-code --function-name $LAMBDA_FUNCTION_NAME --zip-file fileb://$BUILD_DIR/$ZIP_FILE --region $REGION

    if [ $? -ne 0 ]; then
        echo "Failed to update Lambda function"
        exit 1
    fi
else
    # Create a new Lambda function
    echo "Creating Lambda function..."
    aws lambda create-function --function-name $LAMBDA_FUNCTION_NAME \
        --zip-file fileb://$BUILD_DIR/$ZIP_FILE --handler main \
        --runtime provided.al2 --role $ROLE_ARN --region $REGION

    if [ $? -ne 0 ]; then
        echo "Failed to create Lambda function"
        exit 1
    fi
fi

# Clean up
echo "Cleaning up..."
rm -rf $BUILD_DIR

echo "Deployment complete."