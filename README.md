## AWS SDK wrapper for Go

- [dynamo](doc/dynamodb.md)
- [cognito](doc/cognito.md)


## setting the custom domain in SAM

create the certificate in ACM

```
aws acm request-certificate \
  --domain-name subdomain \
  --validation-method DNS \
  --region ap-northeast-1
```

You actually can acm with cloudformation, but I prefer to use the cli due to the condition that I have to wait for the validation.

Next, you build and deploy the SAM application with the custom domain name, hosted zone ID, and certificate ARN:

```
sam build
```

```
sam deploy \
  --parameter-overrides \
  "CustomDomainName=custom_domain \
  HostedZoneId=route53 hostedZoneID \
  CertificateArn=arn_made_with_acm_request-certificate " \
  --capabilities CAPABILITY_IAM
```

## create acm
```
aws acm request-certificate \
  --domain-name domain_name \
  --validation-method DNS \
  --region ap-northeast-1
```