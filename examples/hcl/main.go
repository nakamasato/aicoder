package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func main() {
	// 例: 読み込むTerraformファイルの内容
	tfContent := []byte(`
resource "google_secret_manager_secret_iam_member" "example_sa_is_slack_token_secret_accessor" {
  project   = var.gcp_project_id
  member    = google_service_account.example_sa.member
  secret_id = google_secret_manager_secret.slack_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
}

resource "google_storage_bucket" "example_bucket" {
  name     = "example-bucket"
  location = "US"
}

resource "google_compute_instance" "example_instance" {
  name         = "example-instance"
  machine_type = "n1-standard-1"
  zone         = "us-central1-a"
}
`)

	// 一時ファイルに書き出し
	tmpfile, err := os.CreateTemp("", "example-*.tf")
	if err != nil {
		log.Fatalf("failed to create temp file: %s", err)
	}
	defer os.Remove(tmpfile.Name()) // プログラム終了時にファイルを削除

	// HCL ファイルをパース
	f, diags := hclwrite.ParseConfig(tfContent, tmpfile.Name(), hcl.InitialPos)
	if diags.HasErrors() {
		log.Fatalf("failed to parse HCL: %s", diags.Error())
	}

	// 属性を更新
	log.Println("--- UpdateAttributes ---- example_sa_is_slack_token_secret_accessor")
	file.UpdateAttributes(f, "example_sa_is_slack_token_secret_accessor", map[string]string{
		"role": "roles/secretmanager.secretEditor",
	})

	// ブロックを更新
	log.Println("--- UpdateBlock ---- example_sa_is_slack_token_secret_accessor")
	file.UpdateBlock(f, "resource", "example_sa_is_slack_token_secret_accessor", `
project   = "new_project_id"
member    = "new_member"
secret_id = "new_secret_id"
role      = "roles/secretmanager.admin"
`)

	// ブロックを追加
	log.Println("--- AddBlock ---- example_network")
	file.AddBlock(f, "resource", []string{"google_compute_network", "example_network"}, `  name = "example-network"
  auto_create_subnetworks = true
`)
	// ブロックの名前を更新
	log.Println("--- UpdateResourceNames ---- example_sa_is_slack_token_secret_accessor")
	file.UpdateResourceNames(f, map[string]string{"example_sa_is_slack_token_secret_accessor": "new_resource_name"})

	// 変更後の内容を取得
	modifiedContent := f.Bytes()

	// Save the modified content to a file
	if err := os.WriteFile(tmpfile.Name(), modifiedContent, 0644); err != nil {
		log.Fatalf("failed to write updated file: %s", err)
	}

	// Diffを生成
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(tfContent), string(modifiedContent), false)

	// Diffを表示
	fmt.Println(dmp.DiffPrettyText(diffs))
}
