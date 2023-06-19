source    = ["=artifact="]
bundle_id = "com.komodor.komocli"

apple_id {
  username = "=username="
  password = "=password="
}

sign {
  application_identity = "Developer ID Application: Komodor Automation LTD (F584U99DLC)"
}

zip {
  output_path = "=artifact=.zip"
}