package types

import (
	"fmt"
	"strings"
	"testing"

	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func TestProposalTypeRegistered(t *testing.T) {
	t.Parallel()

	if !gov.IsValidProposalType(ProposalTypeOcrSetConfig) {
		t.Fatalf("proposal type %q is not registered", ProposalTypeOcrSetConfig)
	}
}

func TestSetConfigProposalAminoRegistration(t *testing.T) {
	t.Parallel()

	var content gov.Content = &SetConfigProposal{
		Title:       "title",
		Description: "description",
	}

	bz, err := amino.MarshalJSON(content)
	if err != nil {
		t.Fatalf("marshal gov.Content via amino: %v", err)
	}

	if !strings.Contains(string(bz), "ocr/SetConfigProposal") {
		t.Fatalf("expected amino type tag to include ocr/SetConfigProposal, got: %s", string(bz))
	}
}

func TestLegacyAminoRegistrationPanicsAfterSeal(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when registering on sealed legacy amino codec")
		}
		if !strings.Contains(fmt.Sprint(r), "codec sealed") {
			t.Fatalf("expected panic to contain %q, got: %v", "codec sealed", r)
		}
	}()

	amino.RegisterConcrete(&SetConfigProposal{}, "injective/OcrSetConfigProposal", nil)
}
