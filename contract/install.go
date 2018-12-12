package contract

import (
	reqContext "context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	contextAPI "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/context"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/comm"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/resource"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	en_config "triasVM/config"
)

func (contr *Contract) InstallContract() error {
	ccPkg, err := packager.NewCCPackage(contr.contractPath, "");
	if err != nil {
		return err;
	}
	sdk, err := fabsdk.New(en_config.ConfigBackend)
	defer sdk.Close()
	reqCtx, cancel, err := getContext(sdk, en_config.AdminUser, contr.orgName)
	defer cancel()
	if err != nil {
		return err
	}
	peers, err := getProposalProcessors(sdk, "Admin", en_config.OrdererOrgName, []string{en_config.PeerAddress})
	if err != nil {
		return err;
	}
	if err := installCC(reqCtx, contr.contractName, contr.contractPath, contr.contractVersion, ccPkg, peers); err != nil {
		return err;
	}
	chaincodeQueryResponse, err := resource.QueryInstalledChaincodes(reqCtx, peers[0], resource.WithRetry(retry.DefaultResMgmtOpts))
	if err := installCC(reqCtx, contr.contractName, contr.contractPath, contr.contractVersion, ccPkg, peers); err != nil {
		return err;
	}
	retrieveInstalledCC(chaincodeQueryResponse, contr)
	if err == nil {
		return err;
	}
	//var sdk = *fabsdk.FabricSDK;
	//reqCtx, cancel, err := getContext(sdk, "Admin", org1Name)
	return nil;
}

func retrieveInstalledCC(chaincodeQueryResponse *peer.ChaincodeQueryResponse, contract *Contract) error {
	ccFound := false
	for _, chaincode := range chaincodeQueryResponse.Chaincodes {
		if chaincode.Name == contract.contractName && chaincode.Path == contract.contractPath && chaincode.Version == contract.contractVersion {
			ccFound = true
		}
	}
	if !ccFound {
		return errors.Errorf("Failed to retrieve installed chaincode.");
	}
	return nil
}

func getContext(sdk *fabsdk.FabricSDK, user string, orgName string) (reqContext.Context, reqContext.CancelFunc, error) {

	ctx := sdk.Context(fabsdk.WithUser(user), fabsdk.WithOrg(orgName))

	clientContext, err := ctx()
	if err != nil {
		return nil, nil, errors.WithMessage(err, "create context failed")
	}

	reqCtx, cancel := context.NewRequest(&context.Client{Providers: clientContext, SigningIdentity: clientContext}, context.WithTimeoutType(fab.PeerResponse))
	return reqCtx, cancel, nil
}

func getProposalProcessors(sdk *fabsdk.FabricSDK, user string, orgName string, targets []string) ([]fab.ProposalProcessor, error) {
	ctxProvider := sdk.Context(fabsdk.WithUser(user), fabsdk.WithOrg(orgName))

	ctx, err := ctxProvider()
	if err != nil {
		return nil, errors.WithMessage(err, "context creation failed")
	}

	var peers []fab.ProposalProcessor
	for _, url := range targets {
		p, err := getPeer(ctx, url)
		if err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}

	return peers, nil
}

func getPeer(ctx contextAPI.Client, url string) (fab.Peer, error) {

	peerCfg, err := comm.NetworkPeerConfig(ctx.EndpointConfig(), url)
	if err != nil {
		return nil, err
	}

	peer, err := ctx.InfraProvider().CreatePeerFromConfig(peerCfg)
	if err != nil {
		return nil, errors.WithMessage(err, "creating peer from config failed")
	}

	return peer, nil
}

func installCC(reqCtx reqContext.Context, name string, path string, version string, ccPackage *resource.CCPackage, targets []fab.ProposalProcessor) error {

	icr := resource.InstallChaincodeRequest{Name: name, Path: path, Version: version, Package: ccPackage}

	r, _, err := resource.InstallChaincode(reqCtx, icr, targets, resource.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return errors.WithMessage(err, "InstallChaincode failed")
	}

	// check if response status is not success
	for _, response := range r {
		// return on first not success status
		if response.Status != int32(common.Status_SUCCESS) {
			return errors.Errorf("InstallChaincode returned response status: [%d], cc status: [%d], message: [%s]", response.Status, response.ChaincodeStatus, response.GetResponse().Message)
		}
	}
	return nil
}
