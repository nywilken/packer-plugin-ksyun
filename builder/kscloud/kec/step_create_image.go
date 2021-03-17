package kec

import (
	"context"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateKingcloudImage struct {
	KingcloudRunConfig *KingcloudRunConfig
	KingcloudImageConfig *KingcloudImageConfig
}

func (s *stepCreateKingcloudImage) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	ui := stateBag.Get("ui").(packersdk.Ui)
	client := stateBag.Get("client").(*ClientWrapper)
	instanceId := stateBag.Get("InstanceId").(string)
	ui.Say("Creating Kingcloud Kec Image ")
	createImage := make(map[string]interface{})
	createImage["InstanceId"] = instanceId
	createImage["Name"] = s.KingcloudImageConfig.KingcloudImageName
	resp,errorCreate := client.KecClient.CreateImage(&createImage)
	if errorCreate !=nil {
		return Halt(stateBag, errorCreate, "Error creating  kec image")
	}
	if resp != nil {
		ui.Say("Waiting Kingcloud Kec Image active")
		imageId := getSdkValue(stateBag,"ImageId",*resp).(string)
		_,err := client.WaitKecImageStatus(stateBag,imageId,"active")
		if err !=nil {
			return Halt(stateBag, err, "Error waiting  kec image active")
		}
		stateBag.Put("TargetImageId",imageId)
	}
	return multistep.ActionContinue
}

func (s *stepCreateKingcloudImage) Cleanup(stateBag multistep.StateBag) {

}

