   source = ["./dist/komocli-macos_darwin_amd64/komocli"]
   bundle_id = "com.example.komocli"

   apple_id {
     username = "{{ env "APPLE_ID_USERNAME" }}"
     password = "{{ env "APP_SPECIFIC_PASSWORD" }}"
   }

   sign {
     application_identity = "Developer ID Application: Komodor Automation LTD (F584U99DLC)"
     certificate_base64 = "{{ env "CERTIFICATE_BASE64" }}"
   }

   zip {
     output_path = "komocli-signed.zip"
   }
