package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMachineIdStageOptions(t *testing.T) {
	firstboot := MachineIdFirstBootYes

	expectedOptions := &MachineIdStageOptions{
		FirstBoot: firstboot,
	}

	actualOptions := NewMachineIdStageOptions(firstboot)
	assert.Equal(t, expectedOptions, actualOptions)
}

func TestNewMachineIdStage(t *testing.T) {
	firstboot := MachineIdFirstBootYes

	expectedStage := &Stage{
		Type:    "org.osbuild.machine-id",
		Options: NewMachineIdStageOptions(firstboot),
	}

	actualStage := NewMachineIdStage(NewMachineIdStageOptions(firstboot))
	assert.Equal(t, expectedStage, actualStage)
}
