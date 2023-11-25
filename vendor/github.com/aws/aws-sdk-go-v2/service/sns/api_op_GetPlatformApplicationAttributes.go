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

// Retrieves the attributes of the platform application object for the supported
// push notification services, such as APNS and GCM (Firebase Cloud Messaging). For
// more information, see Using Amazon SNS Mobile Push Notifications (https://docs.aws.amazon.com/sns/latest/dg/SNSMobilePush.html)
// .
func (c *Client) GetPlatformApplicationAttributes(ctx context.Context, params *GetPlatformApplicationAttributesInput, optFns ...func(*Options)) (*GetPlatformApplicationAttributesOutput, error) {
	if params == nil {
		params = &GetPlatformApplicationAttributesInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "GetPlatformApplicationAttributes", params, optFns, c.addOperationGetPlatformApplicationAttributesMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*GetPlatformApplicationAttributesOutput)
	out.ResultMetadata = metadata
	return out, nil
}

// Input for GetPlatformApplicationAttributes action.
type GetPlatformApplicationAttributesInput struct {

	// PlatformApplicationArn for GetPlatformApplicationAttributesInput.
	//
	// This member is required.
	PlatformApplicationArn *string

	noSmithyDocumentSerde
}

// Response for GetPlatformApplicationAttributes action.
type GetPlatformApplicationAttributesOutput struct {

	// Attributes include the following:
	//   - AppleCertificateExpiryDate – The expiry date of the SSL certificate used to
	//   configure certificate-based authentication.
	//   - ApplePlatformTeamID – The Apple developer account ID used to configure
	//   token-based authentication.
	//   - ApplePlatformBundleID – The app identifier used to configure token-based
	//   authentication.
	//   - EventEndpointCreated – Topic ARN to which EndpointCreated event
	//   notifications should be sent.
	//   - EventEndpointDeleted – Topic ARN to which EndpointDeleted event
	//   notifications should be sent.
	//   - EventEndpointUpdated – Topic ARN to which EndpointUpdate event notifications
	//   should be sent.
	//   - EventDeliveryFailure – Topic ARN to which DeliveryFailure event
	//   notifications should be sent upon Direct Publish delivery failure (permanent) to
	//   one of the application's endpoints.
	Attributes map[string]string

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationGetPlatformApplicationAttributesMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsquery_serializeOpGetPlatformApplicationAttributes{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsquery_deserializeOpGetPlatformApplicationAttributes{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "GetPlatformApplicationAttributes"); err != nil {
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
	if err = addOpGetPlatformApplicationAttributesValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opGetPlatformApplicationAttributes(options.Region), middleware.Before); err != nil {
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

func newServiceMetadataMiddleware_opGetPlatformApplicationAttributes(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "GetPlatformApplicationAttributes",
	}
}
