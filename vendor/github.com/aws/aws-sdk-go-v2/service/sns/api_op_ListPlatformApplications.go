// Code generated by smithy-go-codegen DO NOT EDIT.

package sns

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Lists the platform application objects for the supported push notification
// services, such as APNS and GCM (Firebase Cloud Messaging). The results for
// ListPlatformApplications are paginated and return a limited list of
// applications, up to 100. If additional records are available after the first
// page results, then a NextToken string will be returned. To receive the next
// page, you call ListPlatformApplications using the NextToken string received
// from the previous call. When there are no more records to return, NextToken
// will be null. For more information, see [Using Amazon SNS Mobile Push Notifications].
//
// This action is throttled at 15 transactions per second (TPS).
//
// [Using Amazon SNS Mobile Push Notifications]: https://docs.aws.amazon.com/sns/latest/dg/SNSMobilePush.html
func (c *Client) ListPlatformApplications(ctx context.Context, params *ListPlatformApplicationsInput, optFns ...func(*Options)) (*ListPlatformApplicationsOutput, error) {
	if params == nil {
		params = &ListPlatformApplicationsInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "ListPlatformApplications", params, optFns, c.addOperationListPlatformApplicationsMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*ListPlatformApplicationsOutput)
	out.ResultMetadata = metadata
	return out, nil
}

// Input for ListPlatformApplications action.
type ListPlatformApplicationsInput struct {

	// NextToken string is used when calling ListPlatformApplications action to
	// retrieve additional records that are available after the first page results.
	NextToken *string

	noSmithyDocumentSerde
}

// Response for ListPlatformApplications action.
type ListPlatformApplicationsOutput struct {

	// NextToken string is returned when calling ListPlatformApplications action if
	// additional records are available after the first page results.
	NextToken *string

	// Platform applications returned when calling ListPlatformApplications action.
	PlatformApplications []types.PlatformApplication

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationListPlatformApplicationsMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsquery_serializeOpListPlatformApplications{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsquery_deserializeOpListPlatformApplications{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "ListPlatformApplications"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = addClientRequestID(stack); err != nil {
		return err
	}
	if err = addComputeContentLength(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = addComputePayloadSHA256(stack); err != nil {
		return err
	}
	if err = addRetry(stack, options); err != nil {
		return err
	}
	if err = addRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = addRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addSpanRetryLoop(stack, options); err != nil {
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
	if err = addTimeOffsetBuild(stack, c); err != nil {
		return err
	}
	if err = addUserAgentRetryMode(stack, options); err != nil {
		return err
	}
	if err = addCredentialSource(stack, options); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opListPlatformApplications(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addRecursionDetection(stack); err != nil {
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
	if err = addSpanInitializeStart(stack); err != nil {
		return err
	}
	if err = addSpanInitializeEnd(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestStart(stack); err != nil {
		return err
	}
	if err = addSpanBuildRequestEnd(stack); err != nil {
		return err
	}
	return nil
}

// ListPlatformApplicationsPaginatorOptions is the paginator options for
// ListPlatformApplications
type ListPlatformApplicationsPaginatorOptions struct {
	// Set to true if pagination should stop if the service returns a pagination token
	// that matches the most recent token provided to the service.
	StopOnDuplicateToken bool
}

// ListPlatformApplicationsPaginator is a paginator for ListPlatformApplications
type ListPlatformApplicationsPaginator struct {
	options   ListPlatformApplicationsPaginatorOptions
	client    ListPlatformApplicationsAPIClient
	params    *ListPlatformApplicationsInput
	nextToken *string
	firstPage bool
}

// NewListPlatformApplicationsPaginator returns a new
// ListPlatformApplicationsPaginator
func NewListPlatformApplicationsPaginator(client ListPlatformApplicationsAPIClient, params *ListPlatformApplicationsInput, optFns ...func(*ListPlatformApplicationsPaginatorOptions)) *ListPlatformApplicationsPaginator {
	if params == nil {
		params = &ListPlatformApplicationsInput{}
	}

	options := ListPlatformApplicationsPaginatorOptions{}

	for _, fn := range optFns {
		fn(&options)
	}

	return &ListPlatformApplicationsPaginator{
		options:   options,
		client:    client,
		params:    params,
		firstPage: true,
		nextToken: params.NextToken,
	}
}

// HasMorePages returns a boolean indicating whether more pages are available
func (p *ListPlatformApplicationsPaginator) HasMorePages() bool {
	return p.firstPage || (p.nextToken != nil && len(*p.nextToken) != 0)
}

// NextPage retrieves the next ListPlatformApplications page.
func (p *ListPlatformApplicationsPaginator) NextPage(ctx context.Context, optFns ...func(*Options)) (*ListPlatformApplicationsOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.NextToken = p.nextToken

	optFns = append([]func(*Options){
		addIsPaginatorUserAgent,
	}, optFns...)
	result, err := p.client.ListPlatformApplications(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.nextToken
	p.nextToken = result.NextToken

	if p.options.StopOnDuplicateToken &&
		prevToken != nil &&
		p.nextToken != nil &&
		*prevToken == *p.nextToken {
		p.nextToken = nil
	}

	return result, nil
}

// ListPlatformApplicationsAPIClient is a client that implements the
// ListPlatformApplications operation.
type ListPlatformApplicationsAPIClient interface {
	ListPlatformApplications(context.Context, *ListPlatformApplicationsInput, ...func(*Options)) (*ListPlatformApplicationsOutput, error)
}

var _ ListPlatformApplicationsAPIClient = (*Client)(nil)

func newServiceMetadataMiddleware_opListPlatformApplications(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "ListPlatformApplications",
	}
}
