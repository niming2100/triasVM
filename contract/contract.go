package contract

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
	en_config "triasVM/config"
)

type Contract struct {
	peerAddress     string
	contractName    string
	contractType    string
	contractPath    string
	contractVersion string
	channelID       string
	orgName         string
	args            [][]byte
	action          string
}

func NewContract(peerAddress string, contractName string, contractType string, contractPath string, contractVersion string, channelID string, orgName string, args [][]byte, action string) *Contract {
	return &Contract{peerAddress: peerAddress, contractName: contractName, contractType: contractType, contractPath: contractPath, contractVersion: contractVersion, channelID: channelID, orgName: orgName, args: args, action: action}
}

func (c *Contract) Instantiated() (bool, error) {
	sdk, err := fabsdk.New(en_config.ConfigBackend)
	defer sdk.Close()
	if err != nil {
		return false, err;
	}
	instantiated, err := queryInstantiatedCCWithSDK(sdk, fabsdk.WithUser(en_config.AdminUser), c.orgName, c.channelID, c.contractName, en_config.ContractVersion, false)
	return instantiated, nil;
}

func (c *Contract) RunContract() ([]byte, error) {
	sdk, err := fabsdk.New(en_config.ConfigBackend)
	if err != nil {
		return nil, err;
	}
	defer sdk.Close()
	//prepare channel client context using client context
	clientChannelContext := sdk.ChannelContext(c.channelID, fabsdk.WithUser(en_config.AdminUser), fabsdk.WithOrg(c.orgName))

	// Channel client is used to query and execute transactions (Org1 is default org)
	client, err := channel.New(clientChannelContext)

	var resp []byte;
	switch c.action {
	case "query":
		resp, err = c.queryCC(client);
		break;
	case "invoke":
		resp, err = c.executeCC(client);
		break;
	default:
		break;
	}
	return resp, nil;
}

func (c *Contract) executeCC(client *channel.Client) ([]byte, error) {
	response, err := client.Execute(channel.Request{ChaincodeID: c.channelID, Fcn: c.action, Args: c.args},
		channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		fmt.Println("Failed to move funds: %s", err)
		return nil, err
	}
	fmt.Println(response)
	return response.Payload, nil
}

func (c *Contract) queryCC(client *channel.Client) ([]byte, error) {
	response, err := client.Query(channel.Request{ChaincodeID: c.channelID, Fcn: c.action, Args: c.args},
		channel.WithRetry(retry.DefaultChannelOpts))
	if err != nil {
		fmt.Println("Failed to query funds: %s", err)
		return nil, err
	}
	fmt.Println(response)

	return response.Payload, nil
}

func queryInstantiatedCCWithSDK(sdk *fabsdk.FabricSDK, user fabsdk.ContextOption, orgName string, channelID, ccName, ccVersion string, transientRetry bool) (bool, error) {
	clientContext := sdk.Context(user, fabsdk.WithOrg(orgName))

	resMgmt, err := resmgmt.New(clientContext)
	if err != nil {
		return false, errors.WithMessage(err, "Creating resource management client failed")
	}

	return queryInstantiatedCC(resMgmt, orgName, channelID, ccName, ccVersion, transientRetry)
}

func queryInstantiatedCC(resMgmt *resmgmt.Client, orgName string, channelID, ccName, ccVersion string, transientRetry bool) (bool, error) {

	instantiated, err := retry.NewInvoker(retry.New(retry.TestRetryOpts)).Invoke(
		func() (interface{}, error) {
			ok, err := isCCInstantiated(resMgmt, channelID, ccName, ccVersion)
			if err != nil {
				return &ok, err
			}
			if !ok && transientRetry {
				return &ok, status.New(status.TestStatus, status.GenericTransient.ToInt32(), fmt.Sprintf("Did NOT find instantiated chaincode [%s:%s] on one or more peers in [%s].", ccName, ccVersion, orgName), nil)
			}
			return &ok, nil
		},
	)

	if err != nil {
		s, ok := status.FromError(err)
		if ok && s.Code == status.GenericTransient.ToInt32() {
			return false, nil
		}
		return false, errors.WithMessage(err, "isCCInstantiated invocation failed")
	}

	return *instantiated.(*bool), nil
}

func isCCInstantiated(resMgmt *resmgmt.Client, channelID, ccName, ccVersion string) (bool, error) {
	chaincodeQueryResponse, err := resMgmt.QueryInstantiatedChaincodes(channelID, resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return false, errors.WithMessage(err, "Query for instantiated chaincodes failed")
	}

	for _, chaincode := range chaincodeQueryResponse.Chaincodes {
		if chaincode.Name == ccName && chaincode.Version == ccVersion {
			return true, nil
		}
	}
	return false, nil
}
