package sato

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/awslabs/goformation/v7"
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	tftemplate "text/template"
)

type M map[string]interface{}

type Variable struct {
	Description string
	Type        string
	Default     string
}

func Parse(file string, destination string) error {
	// Open a template from file (can be JSON or YAML)
	fileAbs, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	template, err := goformation.Open(fileAbs)
	if err != nil {
		return err
	}

	funcMap := tftemplate.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
		"Deref":   func(str *string) string { return *str },
		"Marshal": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
		"Replace": replace,
		"Tags": func(v []tags.Tag) string {
			var temp string
			for _, item := range v {
				if item.Key != "" {
					temp = temp + "\"" + item.Key + "\"" + "=" + "\"" + item.Value + "\"" + "\n"
				}
			}
			return temp
		},
	}

	err = ParseResources(template.Resources, funcMap, destination)
	if err != nil {
		return err
	}

	err = ParseVariables(template, funcMap, destination)
	if err != nil {
		return err
	}
	return nil
}

func ParseVariables(template *cloudformation.Template, funcMap tftemplate.FuncMap, destination string) error {
	var All string
	for Name, param := range template.Parameters {
		var myVariable Variable

		switch param.Type {
		case "String":
			if param.Default == "false" || param.Default == "true" {
				myVariable.Type = "bool"
			} else {
				myVariable.Type = strings.ToLower(param.Type)
			}

		case "List<AWS::EC2::AvailabilityZone::Name>":
			myVariable.Type = "list(string)"
		default:
			log.Print(param.Type)
		}

		myVariable.Description = strings.Replace(param.Description, "${", "$${", -1)

		switch param.Default.(type) {
		case string:
			_, err := strconv.Atoi(param.Default.(string))
			if err == nil {
				myVariable.Type = "number"
				myVariable.Default = param.Default.(string)
			} else {
				if myVariable.Type == "bool" {
					myVariable.Default = param.Default.(string)
				} else {
					if strings.Contains(param.Default.(string), "=") {
						myVariable = StringToMap(param, myVariable)
					} else {
						myVariable.Default = "\"" + param.Default.(string) + "\""
					}
				}
			}
		case float64:
			myVariable.Type = "number"
			myVariable.Default = fmt.Sprintf("%v", param.Default.(float64))
		case interface{}:
			myVariable.Default = "[]"
		default:
			myVariable.Default = "null"
		}

		var output bytes.Buffer
		tmpl, err := tftemplate.New("test").Funcs(funcMap).Parse(string(variableFile))
		if err != nil {
			return err
		}
		_ = tmpl.Execute(&output, M{
			"variable": myVariable,
			"item":     Name,
		})
		All = All + output.String()
	}
	err := Write(All, destination, "variables")
	if err != nil {
		return err
	}
	return nil
}

func StringToMap(param cloudformation.Parameter, myVariable Variable) Variable {
	temp := strings.Split(param.Default.(string), "=")
	var myMap string
	for n := 0; n < len(temp); n++ {
		if n == 0 {
			myMap = myMap + "{ "
		}
		if n%2 == 0 {
			myMap = myMap + "\"" + temp[n] + "\" = "
		} else {
			myMap = myMap + "\"" + temp[n] + "\""
		}
	}
	myVariable.Default = myMap + "}"
	myVariable.Type = "map(string)"
	return myVariable
}

func ParseResources(resources cloudformation.Resources, funcMap tftemplate.FuncMap, destination string) error {
	for item, resource := range resources {
		var output bytes.Buffer

		myType := resources[item].AWSCloudFormationType()

		TFLookup := map[string]interface{}{
			"AWS::SNS::Topic":                       awsSNSTopic,
			"AWS::IAM::Role":                        awsIamRole,
			"AWS::EC2::Route":                       awsRoute,
			"AWS::EC2::RouteTable":                  awsRouteTable,
			"AWS::EC2::NatGateway":                  awsNatGateway,
			"AWS::EC2::VPCGatewayAttachment":        awsVpnGatewayAttachment,
			"AWS::EC2::NetworkAclEntry":             awsNetworkAclRule,
			"AWS::EC2::NetworkAcl":                  awsNetworkAcl,
			"AWS::EC2::EIP":                         awsEIP,
			"AWS::EC2::SubnetRouteTableAssociation": awsRouteTableAssociation,
			"AWS::EC2::Subnet":                      awsSubnet,
			"AWS::Logs::LogGroup":                   awsCloudwatchLogGroup,
			"AWS::EC2::VPCDHCPOptionsAssociation":   awsVpcDhcpOptionsAssociation,
			"AWS::EC2::DHCPOptions":                 awsVpcDhcpOptions,
			"AWS::EC2::SubnetNetworkAclAssociation": awsNetworkAclAssociation,
			"AWS::EC2::FlowLog":                     awsFlowLog,
			"AWS::EC2::VPCEndpoint":                 awsVpcEndpoint,
			"AWS::EC2::InternetGateway":             awsInternetGateway,
			"AWS::EC2::VPC":                         awsVpc,
		}

		var myContent []byte
		if TFLookup[myType] != nil {
			myContent = TFLookup[myType].([]byte)
		} else {
			log.Printf("%s not found", myType)
			continue
		}

		//needs to pivot on policy template from resource
		tmpl, err := tftemplate.New("test").Funcs(funcMap).Parse(string(myContent))

		if err != nil {
			return err
		}

		_ = tmpl.Execute(&output, M{
			"resource": resource,
			"item":     item,
		})
		err = Write(output.String(), destination, fmt.Sprint(ToTFName(myType), ".", strings.ToLower(item)))
		if err != nil {
			return err
		}
	}
	return nil
}

// Write out terraform
func Write(output string, location string, name string) error {

	newPath, _ := filepath.Abs(location)
	err := os.MkdirAll(newPath, os.ModePerm)
	if err != nil {
		return err
	}

	d1 := []byte(output)

	destination, _ := filepath.Abs(fmt.Sprint(location, "/", name, ".tf"))
	err = os.WriteFile(destination, d1, 0644)

	if err != nil {
		return err
	}

	return nil
}

func ToTFName(CFN string) string {
	return strings.ToLower(strings.ReplaceAll(CFN, "::", "_"))
}

func replace(input, from, to string) string {
	return strings.Replace(input, from, to, -1)
}
