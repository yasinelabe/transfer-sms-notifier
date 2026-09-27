package processor

import "testing"

func TestRenderTemplate(t *testing.T) {
	got := RenderTemplate("Transfer {{transferId}} from {{sender}} to {{receiver}} on {{createdDate}}", TemplateVars{
		TransferID:  "15862087576",
		Sender:      "252639339979",
		Receiver:    "252634872297",
		CreatedDate: "2026-09-27T13:09:16Z",
	})
	want := "Transfer 15862087576 from 252639339979 to 252634872297 on 2026-09-27T13:09:16Z"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
