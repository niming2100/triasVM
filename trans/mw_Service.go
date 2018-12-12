package trans

import (
	"fmt"
	enConf "triasVM/config"
	"triasVM/contract"
	"triasVM/proto/tm"
	futils "triasVM/utils"
	"triasVM/validate"
)

func NewMWService() *MWservice {
	return &MWservice{}
}

const (
	pathPrefix = "./source/contract/"
	fileSuffix = ".go"
)

type MWservice struct {
}

func (this *MWservice) ExecuteContract(request *tm.ExecuteContractRequest) (*tm.ExecuteContractResponse, error) {
	// TODO validate
	isCorect, err := validate.RequestValidate(request);
	if !isCorect || err != nil {
		fmt.Println("Contract validate fails", err);
	}
	// TODO CheckContract is install
	var filePath = pathPrefix + request.GetContractName() + fileSuffix;
	isExists, err := futils.PathExists(filePath);
	if err != nil {
		fmt.Println("checkFilePathFails", err);
	}

	contract := contract.NewContract(enConf.PeerAddress, request.GetContractName(), request.GetContractType(), filePath, enConf.ContractVersion, enConf.ChannelID, enConf.OrdererOrgName, request.GetCommand());
	if !isExists {
		//TODO downloadFile and install
		//download file
		err := futils.FileDownLoad(filePath, request.GetAddress());
		if err != nil {
			fmt.Println("Download contract happens a error", err);
		}
		installErr := contract.InstallContract();
		if installErr != nil {
			fmt.Println("Failed to package contract", installErr);
		}
	} else {
		instantiated, err := contract.Instantiated();
		if err != nil {
			fmt.Println("check instantiated happens a error", err);
			return nil,nil;
		}
		if !instantiated {
			installErr := contract.InstallContract();
			if installErr != nil {
				fmt.Println("Failed to package contract", installErr);
			}
		}
	}
	contract.RunContract()
	return nil, nil;
}
