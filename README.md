## Setup

Initialize the DynamoDB ORM structure by creating a DynamoDB client and instantiating the repository. For example:

```go
import (
    "context"
    "log"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    db "github.com/yuki5155/go-aws/dynamodb"
)

func main() {
    cfg, err := config.LoadDefaultConfig(
        context.TODO(),
        config.WithRegion("your_region"),
    )
    if err != nil {
        log.Fatalf("unable to load SDK config: %v", err)
    }
    client := dynamodb.NewFromConfig(cfg)
    repo := db.NewRepository(client, "table_name")

    // Now you can use the repo for CRUD operations.
}
```

---

## Create

Define your data structure with the appropriate tags. For instance, here’s a sample `User` struct. Note that the `dynamo` tag specifies the attribute name, key type, and required fields. It also implements `TableNamer` to override the default table name.

```go
package main

type User struct {
    ID        string `json:"id" dynamodbav:"id" dynamo:"id,key=hash"`
    Email     string `json:"email" dynamodbav:"email" dynamo:"email,required,index=email-index"`
    Name      string `json:"name" dynamodbav:"name" dynamo:"name,required"`
    CreatedAt int64  `json:"created_at" dynamodbav:"created_at" dynamo:"created_at"`
}

// TableName returns the table name to override the default repository table name.
func (u *User) TableName() string {
    return "Users"
}
```

Create a record in DynamoDB by calling the repository’s `Create` method:

```go
err := repo.Create(context.Background(), userInstance)
if err != nil {
    log.Fatalf("unable to create record: %v", err)
}
```

---

## Get

The same data structure is used to retrieve records. You can use either the hash key directly or a global secondary index (GSI) to query records.

### Find by ID

Use the `FindByID` method to retrieve a record using its hash key:

```go
var user User
err := repo.FindByID(context.Background(), "your_id", &user)
if err != nil {
    log.Fatalf("failed to find record by ID: %v", err)
}
```

### Global Secondary Index

When querying by another parameter (e.g., email) that has a GSI defined, use the `FindByParameter` method:

```go
var users []User
err := repo.FindByParameter(context.Background(), "email", "your_email@example.com", &users)
if err != nil {
    log.Fatalf("failed to find record by parameter: %v", err)
}
```

---

## Update

You can update an existing record using the `Update` method. The Update operation:
1. Uses reflection to extract the key field (identified by `key=hash` in the `dynamo` tag).
2. Builds an update expression for the non-key fields.
3. Uses a condition expression to ensure the item exists.

For example, to update a user's email and name:

```go
userToUpdate := User{
    ID:    "your_id", // This is the primary hash key.
    Email: "updated_email@example.com",
    Name:  "Updated Name",
    // Other fields (e.g., CreatedAt) can be updated as needed.
}

err = repo.Update(context.Background(), &userToUpdate)
if err != nil {
    log.Fatalf("failed to update record: %v", err)
}
```

If the record does not exist, the Update method returns an error (for example, ErrNotFound). Also, if no updatable fields are found, it will return an error.

---

## Delete

Delete an item from DynamoDB by its primary key id (assumed to be a string). The `Delete` method uses a conditional expression to ensure that the item exists before attempting deletion.

```go
err = repo.Delete(context.Background(), "your_id")
if err != nil {
    log.Fatalf("failed to delete record: %v", err)
}
```

In this example, the primary key attribute is assumed to have the name "id". If the specified item doesn’t exist, the method returns an error (e.g., ErrNotFound).



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