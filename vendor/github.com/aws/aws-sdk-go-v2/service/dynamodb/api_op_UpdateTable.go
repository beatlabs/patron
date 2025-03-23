// Code generated by smithy-go-codegen DO NOT EDIT.

package dynamodb

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	internalEndpointDiscovery "github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Modifies the provisioned throughput settings, global secondary indexes, or
// DynamoDB Streams settings for a given table.
//
// For global tables, this operation only applies to global tables using Version
// 2019.11.21 (Current version).
//
// You can only perform one of the following operations at once:
//
//   - Modify the provisioned throughput settings of the table.
//
//   - Remove a global secondary index from the table.
//
//   - Create a new global secondary index on the table. After the index begins
//     backfilling, you can use UpdateTable to perform other operations.
//
// UpdateTable is an asynchronous operation; while it's executing, the table
// status changes from ACTIVE to UPDATING . While it's UPDATING , you can't issue
// another UpdateTable request. When the table returns to the ACTIVE state, the
// UpdateTable operation is complete.
func (c *Client) UpdateTable(ctx context.Context, params *UpdateTableInput, optFns ...func(*Options)) (*UpdateTableOutput, error) {
	if params == nil {
		params = &UpdateTableInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "UpdateTable", params, optFns, c.addOperationUpdateTableMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*UpdateTableOutput)
	out.ResultMetadata = metadata
	return out, nil
}

// Represents the input of an UpdateTable operation.
type UpdateTableInput struct {

	// The name of the table to be updated. You can also provide the Amazon Resource
	// Name (ARN) of the table in this parameter.
	//
	// This member is required.
	TableName *string

	// An array of attributes that describe the key schema for the table and indexes.
	// If you are adding a new global secondary index to the table,
	// AttributeDefinitions must include the key element(s) of the new index.
	AttributeDefinitions []types.AttributeDefinition

	// Controls how you are charged for read and write throughput and how you manage
	// capacity. When switching from pay-per-request to provisioned capacity, initial
	// provisioned capacity values must be set. The initial provisioned capacity values
	// are estimated based on the consumed read and write capacity of your table and
	// global secondary indexes over the past 30 minutes.
	//
	//   - PROVISIONED - We recommend using PROVISIONED for predictable workloads.
	//   PROVISIONED sets the billing mode to [Provisioned capacity mode].
	//
	//   - PAY_PER_REQUEST - We recommend using PAY_PER_REQUEST for unpredictable
	//   workloads. PAY_PER_REQUEST sets the billing mode to [On-demand capacity mode].
	//
	// [Provisioned capacity mode]: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/provisioned-capacity-mode.html
	// [On-demand capacity mode]: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/on-demand-capacity-mode.html
	BillingMode types.BillingMode

	// Indicates whether deletion protection is to be enabled (true) or disabled
	// (false) on the table.
	DeletionProtectionEnabled *bool

	// An array of one or more global secondary indexes for the table. For each index
	// in the array, you can request one action:
	//
	//   - Create - add a new global secondary index to the table.
	//
	//   - Update - modify the provisioned throughput settings of an existing global
	//   secondary index.
	//
	//   - Delete - remove a global secondary index from the table.
	//
	// You can create or delete only one global secondary index per UpdateTable
	// operation.
	//
	// For more information, see [Managing Global Secondary Indexes] in the Amazon DynamoDB Developer Guide.
	//
	// [Managing Global Secondary Indexes]: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/GSI.OnlineOps.html
	GlobalSecondaryIndexUpdates []types.GlobalSecondaryIndexUpdate

	// Specifies the consistency mode for a new global table. This parameter is only
	// valid when you create a global table by specifying one or more [Create]actions in the [ReplicaUpdates]
	// action list.
	//
	// You can specify one of the following consistency modes:
	//
	//   - EVENTUAL : Configures a new global table for multi-Region eventual
	//   consistency. This is the default consistency mode for global tables.
	//
	//   - STRONG : Configures a new global table for multi-Region strong consistency
	//   (preview).
	//
	// Multi-Region strong consistency (MRSC) is a new DynamoDB global tables
	//   capability currently available in preview mode. For more information, see [Global tables multi-Region strong consistency].
	//
	// If you don't specify this parameter, the global table consistency mode defaults
	// to EVENTUAL .
	//
	// [ReplicaUpdates]: https://docs.aws.amazon.com/https:/docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_UpdateTable.html#DDB-UpdateTable-request-ReplicaUpdates
	// [Create]: https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_ReplicationGroupUpdate.html#DDB-Type-ReplicationGroupUpdate-Create
	// [Global tables multi-Region strong consistency]: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/PreviewFeatures.html#multi-region-strong-consistency-gt
	MultiRegionConsistency types.MultiRegionConsistency

	// Updates the maximum number of read and write units for the specified table in
	// on-demand capacity mode. If you use this parameter, you must specify
	// MaxReadRequestUnits , MaxWriteRequestUnits , or both.
	OnDemandThroughput *types.OnDemandThroughput

	// The new provisioned throughput settings for the specified table or index.
	ProvisionedThroughput *types.ProvisionedThroughput

	// A list of replica update actions (create, delete, or update) for the table.
	//
	// For global tables, this property only applies to global tables using Version
	// 2019.11.21 (Current version).
	ReplicaUpdates []types.ReplicationGroupUpdate

	// The new server-side encryption settings for the specified table.
	SSESpecification *types.SSESpecification

	// Represents the DynamoDB Streams configuration for the table.
	//
	// You receive a ValidationException if you try to enable a stream on a table that
	// already has a stream, or if you try to disable a stream on a table that doesn't
	// have a stream.
	StreamSpecification *types.StreamSpecification

	// The table class of the table to be updated. Valid values are STANDARD and
	// STANDARD_INFREQUENT_ACCESS .
	TableClass types.TableClass

	// Represents the warm throughput (in read units per second and write units per
	// second) for updating a table.
	WarmThroughput *types.WarmThroughput

	noSmithyDocumentSerde
}

func (in *UpdateTableInput) bindEndpointParams(p *EndpointParameters) {

	p.ResourceArn = in.TableName

}

// Represents the output of an UpdateTable operation.
type UpdateTableOutput struct {

	// Represents the properties of the table.
	TableDescription *types.TableDescription

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationUpdateTableMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson10_serializeOpUpdateTable{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson10_deserializeOpUpdateTable{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "UpdateTable"); err != nil {
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
	if err = addOpUpdateTableDiscoverEndpointMiddleware(stack, options, c); err != nil {
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
	if err = addUserAgentAccountIDEndpointMode(stack, options); err != nil {
		return err
	}
	if err = addCredentialSource(stack, options); err != nil {
		return err
	}
	if err = addOpUpdateTableValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opUpdateTable(options.Region), middleware.Before); err != nil {
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
	if err = addValidateResponseChecksum(stack, options); err != nil {
		return err
	}
	if err = addAcceptEncodingGzip(stack, options); err != nil {
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

func addOpUpdateTableDiscoverEndpointMiddleware(stack *middleware.Stack, o Options, c *Client) error {
	return stack.Finalize.Insert(&internalEndpointDiscovery.DiscoverEndpoint{
		Options: []func(*internalEndpointDiscovery.DiscoverEndpointOptions){
			func(opt *internalEndpointDiscovery.DiscoverEndpointOptions) {
				opt.DisableHTTPS = o.EndpointOptions.DisableHTTPS
				opt.Logger = o.Logger
			},
		},
		DiscoverOperation:            c.fetchOpUpdateTableDiscoverEndpoint,
		EndpointDiscoveryEnableState: o.EndpointDiscovery.EnableEndpointDiscovery,
		EndpointDiscoveryRequired:    false,
		Region:                       o.Region,
	}, "ResolveEndpointV2", middleware.After)
}

func (c *Client) fetchOpUpdateTableDiscoverEndpoint(ctx context.Context, region string, optFns ...func(*internalEndpointDiscovery.DiscoverEndpointOptions)) (internalEndpointDiscovery.WeightedAddress, error) {
	input := getOperationInput(ctx)
	in, ok := input.(*UpdateTableInput)
	if !ok {
		return internalEndpointDiscovery.WeightedAddress{}, fmt.Errorf("unknown input type %T", input)
	}
	_ = in

	identifierMap := make(map[string]string, 0)
	identifierMap["sdk#Region"] = region

	key := fmt.Sprintf("DynamoDB.%v", identifierMap)

	if v, ok := c.endpointCache.Get(key); ok {
		return v, nil
	}

	discoveryOperationInput := &DescribeEndpointsInput{}

	opt := internalEndpointDiscovery.DiscoverEndpointOptions{}
	for _, fn := range optFns {
		fn(&opt)
	}

	go c.handleEndpointDiscoveryFromService(ctx, discoveryOperationInput, region, key, opt)
	return internalEndpointDiscovery.WeightedAddress{}, nil
}

func newServiceMetadataMiddleware_opUpdateTable(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "UpdateTable",
	}
}
