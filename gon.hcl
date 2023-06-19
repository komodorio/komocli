source    = ["./dist/komocli_darwin_amd64/komocli"]
bundle_id = "com.example.komocli"

apple_id {
  username = "{{ username }}"
  password = "{{ password }}"
}

sign {
  application_identity = "Developer ID Application: Komodor Automation LTD (F584U99DLC)"
}

zip {
  output_path = "komocli-signed.zip"
}
