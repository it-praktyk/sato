package sato

import (
	"bytes"
	"fmt"
	"strings"
	tftemplate "text/template"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/rs/zerolog/log"
)

// ParseResources converts resource to Terraform
func ParseResources(resources cloudformation.Resources, funcMap tftemplate.FuncMap, destination string) error {
	for item, resource := range resources {
		var output bytes.Buffer

		myType := resources[item].AWSCloudFormationType()

		myContent := lookup(myType)

		//needs to pivot on policy template from resource
		tmpl, err := tftemplate.New("sato").Funcs(funcMap).Parse(string(myContent))

		if err != nil {
			return err
		}

		_ = tmpl.Execute(&output, M{
			"resource": resource,
			"item":     item,
		})

		err = Write(ReplaceDependant(ReplaceVariables(output.String())), destination, fmt.Sprint(ToTFName(myType), ".", strings.ToLower(item)))
		if err != nil {
			return err
		}
	}
	return nil
}

func lookup(myType string) []byte {
	TFLookup := map[string]interface{}{
		"AWS::SNS::Topic":                                 awsSNSTopic,
		"AWS::IAM::Role":                                  awsIamRole,
		"AWS::EC2::Route":                                 awsRoute,
		"AWS::EC2::RouteTable":                            awsRouteTable,
		"AWS::EC2::NatGateway":                            awsNatGateway,
		"AWS::EC2::VPCGatewayAttachment":                  awsVpnGatewayAttachment,
		"AWS::EC2::NetworkAclEntry":                       awsNetworkACLRule,
		"AWS::EC2::NetworkAcl":                            awsNetworkACL,
		"AWS::EC2::EIP":                                   awsEIP,
		"AWS::EC2::SubnetRouteTableAssociation":           awsRouteTableAssociation,
		"AWS::EC2::Subnet":                                awsSubnet,
		"AWS::Logs::LogGroup":                             awsCloudwatchLogGroup,
		"AWS::EC2::VPCDHCPOptionsAssociation":             awsVpcDhcpOptionsAssociation,
		"AWS::EC2::DHCPOptions":                           awsVpcDhcpOptions,
		"AWS::EC2::SubnetNetworkAclAssociation":           awsNetworkACLAssociation,
		"AWS::EC2::FlowLog":                               awsFlowLog,
		"AWS::EC2::VPCEndpoint":                           awsVpcEndpoint,
		"AWS::EC2::InternetGateway":                       awsInternetGateway,
		"AWS::EC2::VPC":                                   awsVpc,
		"AWS::S3::Bucket":                                 awsS3Bucket,
		"AWS::Lambda::Function":                           awsLambdaFunction,
		"AWS::StepFunctions::StateMachine":                awsStepfunctionStateMachine,
		"AWS::DynamoDB::Table":                            awsDynamodbTable,
		"AWS::IAM::InstanceProfile":                       awsIamInstanceProfile,
		"AWS::CloudFormation::Stack":                      awsCloudformationStack,
		"AWS::EC2::SecurityGroup":                         awsSecurityGroup,
		"AWS::SecretsManager::Secret":                     awsSecretsManagerSecret,
		"AWS::EC2::Instance":                              awsInstance,
		"AWS::S3::BucketPolicy":                           awsS3BucketPolicy,
		"AWS::IAM::ManagedPolicy":                         awsIamManagedPolicy,
		"AWS::IAM::Policy":                                awsIamPolicy,
		"AWS::KMS::Key":                                   awsKmsKey,
		"AWS::KMS::Alias":                                 awskmsAlias,
		"AWS::SSM::Association":                           awsSsmAssociation,
		"AWS::SSM::Document":                              awsSsmDocument,
		"AWS::AutoScaling::LaunchConfiguration":           awsLaunchConfiguration,
		"AWS::AutoScaling::AutoScalingGroup":              awsAutoscalingGroup,
		"AWS::Lambda::Permission":                         awsLambdaPermission,
		"AWS::ElastiCache::SubnetGroup":                   awsElasticacheSubnetGroup,
		"AWS::ElastiCache::ParameterGroup":                awsElasticacheParameterGroup,
		"AWS::ElastiCache::ReplicationGroup":              awsElasticacheReplicationGroup,
		"AWS::DirectoryService::MicrosoftAD":              awsDirectoryServiceDirectory,
		"AWS::CodeBuild::Project":                         awsCodebuildProject,
		"AWS::CodePipeline::Pipeline":                     awsCodepipeline,
		"AWS::EC2::SecurityGroupIngress":                  awsSecurityGroupRuleIngress,
		"AWS::EC2::SecurityGroupEgress":                   awsSecurityGroupRuleEgress,
		"AWS::CloudFront::Distribution":                   awsCloudfrontDistribution,
		"AWS::ElasticLoadBalancingV2::LoadBalancer":       awsLb,
		"AWS::ElasticLoadBalancingV2::ListenerRule":       awsLbListenerRule,
		"AWS::ElasticLoadBalancingV2::TargetGroup":        awsLbTargetGroup,
		"AWS::ElasticLoadBalancingV2::Listener":           awsLbListener,
		"AWS::IAM::User":                                  awsIamUser,
		"AWS::Cloud9::EnvironmentEC2":                     awsCloud9EnvironmentEc2,
		"AWS::CodeCommit::Repository":                     awsCodecommitRepository,
		"AWS::CloudWatch::Alarm":                          awsCloudwatchMetricAlarm,
		"AWS::Route53::RecordSet":                         awsRoute53Record,
		"AWS::RDS::DBSubnetGroup":                         awsDbSubnetGroup,
		"AWS::RDS::DBCluster":                             awsRdsCluster,
		"AWS::Events::Rule":                               awsCloudwatchEventRule,
		"AWS::CloudFront::CloudFrontOriginAccessIdentity": awsCloudfrontOriginAccessIdentity,
		"AWS::ElasticLoadBalancing::LoadBalancer":         awsElb,
		"AWS::AutoScaling::ScalingPolicy":                 awsAutoscalingPolicy,
		"AWS::AutoScaling::ScheduledAction":               awsAutoscalingSchedule,
		"AWS::SNS::TopicPolicy":                           awsSNSTopicPolicy,
		"AWS::Config::DeliveryChannel":                    awsConfigDeliveryChannel,
		"AWS::EC2::Volume":                                awsEbsVolume,
		"AWS::Config::ConfigurationRecorder":              awsConfigConfigurationRecorder,
		"AWS::Config::ConfigRule":                         awsConfigConfigRule,
	}

	var myContent []byte
	if TFLookup[myType] != nil {
		myContent = TFLookup[myType].([]byte)
	} else {
		//we don't want to half the parsing so just log it.
		log.Warn().Msgf("%s not found", myType)
	}
	return myContent
}
