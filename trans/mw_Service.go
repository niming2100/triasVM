package trans

import (
	"encoding/json"
	"fmt"
	"github.com/errors"
	enConf "triasVM/config"
	"triasVM/contract"
	"triasVM/proto/tm"
	t_utils "triasVM/utils"
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
	isExists, err := t_utils.PathExists(filePath);
	if err != nil {
		fmt.Println("checkFilePathFails", err);
	}
	var data map[string][]string;
	if err := json.Unmarshal([]byte(request.GetCommand()), &data); err == nil {
	} else {
		return nil, err
	}
	stringArray := data["Args"];
	if stringArray == nil || len(stringArray) == 0 {
		return nil, errors.Errorf("command illegal")
	}
	args := t_utils.StringArrayToByte(stringArray);
	contract := contract.NewContract(enConf.PeerAddress, request.GetContractName(), request.GetContractType(), filePath, enConf.ContractVersion, enConf.ChannelID, enConf.OrdererOrgName, args, request.GetOpration());
	if !isExists {
		//TODO downloadFile and install
		//download file
		err := t_utils.FileDownLoad(filePath, request.GetAddress());
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
			return nil, nil;
		}
		if !instantiated {
			installErr := contract.InstallContract();
			if installErr != nil {
				fmt.Println("Failed to package contract", installErr);
			}
		}
	}
	resp, err := contract.RunContract()

	return nil, nil;
}
