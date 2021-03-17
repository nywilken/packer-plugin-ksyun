package kec

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"strconv"
)

type stepConfigKingcloudPublicIp struct {
	KingcloudRunConfig *KingcloudRunConfig
	eipId string
}

func (s *stepConfigKingcloudPublicIp) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	ui := stateBag.Get("ui").(packersdk.Ui)
	client := stateBag.Get("client").(*ClientWrapper)
	instanceId := stateBag.Get("InstanceId").(string)
	chargeTypes := []string{"Daily","TrafficMonthly","DailyPaidByTransfer","HourlyInstantSettlement"}
	checkChargeType := false
	if s.KingcloudRunConfig.AssociatePublicIpAddress {
		if s.KingcloudRunConfig.PublicIpBandWidth == 0 {
			// default bandwidth is 1 m
			s.KingcloudRunConfig.PublicIpBandWidth = 1
		}else if s.KingcloudRunConfig.PublicIpBandWidth > 100{
			return Halt(stateBag,fmt.Errorf("public_ip max bandwidth must lower than 100"),"")
		}
		if s.KingcloudRunConfig.PublicIpChargeType == ""{
			// default PublicIpChargeType is Daily
			s.KingcloudRunConfig.PublicIpChargeType = "Daily"
			checkChargeType = true
		}else {
			for _,v:= range chargeTypes{
				if s.KingcloudRunConfig.PublicIpChargeType == v{
					checkChargeType = true
				}
			}
		}
		if checkChargeType {
			ui.Say("Allocating eip...")
			//create eip
			createEip := make(map[string]interface{})
			createEip["BandWidth"] = strconv.Itoa(s.KingcloudRunConfig.PublicIpBandWidth)
			createEip["ChargeType"] = s.KingcloudRunConfig.PublicIpChargeType
			createEip["ProjectId"] = s.KingcloudRunConfig.ProjectId
			createResp,createErr := client.EipClient.AllocateAddress(&createEip)
			if createErr != nil{
				return Halt(stateBag, createErr, "Error creating new eip")
			}
			if createResp != nil{
				allocationId := getSdkValue(stateBag,"AllocationId",*createResp).(string)
				publicIp := getSdkValue(stateBag,"PublicIp",*createResp).(string)
				s.eipId = allocationId
				stateBag.Put("publicIp",publicIp)
				ui.Say("Associating eip to instance")
				//create_security_group_rule
				authorizeSecurityGroupEntry := make(map[string]interface{})
				authorizeSecurityGroupEntry["SecurityGroupId"] = s.KingcloudRunConfig.SecurityGroupId
				authorizeSecurityGroupEntry["CidrBlock"] = "0.0.0.0/0"
				authorizeSecurityGroupEntry["Direction"] = "in"
				authorizeSecurityGroupEntry["Protocol"] = "tcp"
				authorizeSecurityGroupEntry["PortRangeFrom"] = strconv.Itoa(22)
				authorizeSecurityGroupEntry["PortRangeTo"] = strconv.Itoa(22)
				_,errRule := client.VpcClient.AuthorizeSecurityGroupEntry(&authorizeSecurityGroupEntry)
				if errRule != nil {
					return Halt(stateBag, errRule, "Error creating  eip SecurityGroupRule")
				}
				//associate eip
				associateAddress := make(map[string]interface{})
				associateAddress["AllocationId"] = allocationId
				associateAddress["InstanceType"] = "Ipfwd"
				associateAddress["InstanceId"] = instanceId
				_,err := client.EipClient.AssociateAddress(&associateAddress)
				if err != nil{
					return Halt(stateBag, err, "Error associate eip to instance")
				}
			}

		}else{
			return Halt(stateBag,fmt.Errorf("public_ip_charge_type not match"),"")
		}
	}
	return multistep.ActionContinue
}

func (s *stepConfigKingcloudPublicIp) Cleanup(stateBag multistep.StateBag) {
	if s.eipId != "" {
		ui := stateBag.Get("ui").(packersdk.Ui)
		client := stateBag.Get("client").(*ClientWrapper)
		ui.Say(fmt.Sprintf("Disassociate Eip with Id %s ",s.eipId))
		disassociateEip := make(map[string]interface{})
		disassociateEip["AllocationId"] = s.eipId
		_,disassociateErr := client.EipClient.DisassociateAddress(&disassociateEip)
		if disassociateErr != nil {
			ui.Error(fmt.Sprintf("Error disassociate Eip %s", disassociateErr))
		}
		ui.Say(fmt.Sprintf("Deleting Eip with Id %s ",s.eipId))
		deleteEip := make(map[string]interface{})
		deleteEip["AllocationId"] = s.eipId
		_,err := client.EipClient.ReleaseAddress(&deleteEip)
		if err != nil {
			ui.Error(fmt.Sprintf("Error delete Eip %s", err))
		}
	}
}



