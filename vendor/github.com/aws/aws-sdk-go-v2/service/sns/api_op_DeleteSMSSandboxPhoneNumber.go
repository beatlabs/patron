// Code generated by smithy-go-codegen DO NOT EDIT.

package sns

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Deletes an Amazon Web Services account's verified or pending phone number from
// the SMS sandbox. When you start using Amazon SNS to send SMS messages, your
// Amazon Web Services account is in the SMS sandbox. The SMS sandbox provides a
// safe environment for you to try Amazon SNS features without risking your
// reputation as an SMS sender. While your Amazon Web Services account is in the
// SMS sandbox, you can use all of the features of Amazon SNS. However, you can
// send SMS messages only to verified destination phone numbers. For more
// information, including how to move out of the sandbox to send messages without
// restrictions, see SMS sandbox (https://docs.aws.amazon.com/sns/latest/dg/sns-sms-sandbox.html)
// in the Amazon SNS Developer Guide.
func (c *Client) DeleteSMSSandboxPhoneNumber(ctx context.Context, params *DeleteSMSSandboxPhoneNumberInput, optFns ...func(*Options)) (*DeleteSMSSandboxPhoneNumberOutput, error) {
	if params == nil {
		params = &DeleteSMSSandboxPhoneNumberInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "DeleteSMSSandboxPhoneNumber", params, optFns, c.addOperationDeleteSMSSandboxPhoneNumberMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*DeleteSMSSandboxPhoneNumberOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type DeleteSMSSandboxPhoneNumberInput struct {

	// The destination phone number to delete.
	//
	// This member is required.
	PhoneNumber *string

	noSmithyDocumentSerde
}

type DeleteSMSSandboxPhoneNumberOutput struct {
	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationDeleteSMSSandboxPhoneNumberMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsquery_serializeOpDeleteSMSSandboxPhoneNumber{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsquery_deserializeOpDeleteSMSSandboxPhoneNumber{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "DeleteSMSSandboxPhoneNumber"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddClientRequestIDMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddComputeContentLengthMiddleware(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = v4.AddComputePayloadSHA256Middleware(stack); err != nil {
		return err
	}
	if err = addRetryMiddlewares(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addOpDeleteSMSSandboxPhoneNumberValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opDeleteSMSSandboxPhoneNumber(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opDeleteSMSSandboxPhoneNumber(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "DeleteSMSSandboxPhoneNumber",
	}
}
