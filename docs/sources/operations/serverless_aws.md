---
title: Backend search - serverless AWS setup
alias: 
- /docs/tempo/latest/operations/backend_search/serverless_aws/
- /docs/tempo/latest/operations/serverless_aws/
weight: 11
---

# AWS Lambda for serverless backend search

This document will walk you through setting up an AWS Lambda for serverless backend search.
For more guidance on configuration options for full backend search [check here]({{< relref "backend_search/" >}}).

1. Build the code package:

    ```bash
    cd ./cmd/tempo-serverless && make build-lambda-zip
    ```

    This will create a ZIP file containing the binary required for 
    the function. The file name will be of the form: `./lambda/tempo-<branch name>-<commit hash>.zip`.
    Here is an example of that name:

    ```bash
    ls lambda/*.zip
    lambda/tempo-serverless-backend-search-297172a.zip
    ```

2. Provision an S3 bucket.

3. Copy the ZIP file into your bucket.

    ```
    aws s3 cp lambda/tempo-serverless-backend-search-297172a.zip gs://<newly provisioned gcs bucket>
    ```

4. Provision the Lambda. For a Lambda function to be invoked via HTTP, we also need to create an
   ALB and some other resources. This example uses Terraform and only includes the function definition.
   Additionally, you will need a VPC, security groups, IAM roles, an ALB, target groups, etc., but that is
   beyond the scope of this guide.

    ```
    locals {
      // this can be increased if you would like to use multiple functions
      count = 1
    }

    resource "aws_lambda_function" "lambda" {
        count = local.count

        function_name    = "${local.name}-${count.index}"
        description      = "${local.name}-${count.index}"
        role             = <arn of appropriate role>
        handler          = "main"
        runtime          = "go1.x"
        timeout          = "60"
        s3_key           = "tempo-serverless-backend-search-297172a.zip"
        s3_bucket        = <S3 bucket created above>
        memory_size      = 1769 # 1 vcpu

        vpc_config {
            subnet_ids         = <appropriate subnets>
            security_group_ids = [<appropriate security groups>]
        }

        environment {
            variables = {
            "TEMPO_S3_BUCKET"               = "<S3 bucket name backing your Tempo instance>"
            "TEMPO_BACKEND"                 = "s3"
            "TEMPO_S3_HEDGE_REQUESTS_AT"    = "400ms"
            "TEMPO_S3_HEDGE_REQUESTS_UP_TO" = "2"
            "TEMPO_S3_ENDPOINT"             = "s3.dualstack.us-east-2.amazonaws.com"
            }
        }
    }
    ```
    
5. Add the hostname of the newly-created ALB to the querier config:

    ```
    querier:
      search:
        external_endpoints:
        - http://<alb dns hostname>
    ```
