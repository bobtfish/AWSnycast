#!/usr/bin/ruby
require 'json'

output = {
  "variable" => {
    "deploy_ssh_pubkey" => {
      "description" => "The Deployment ssh pub key",
      "default" => IO.read(File.dirname(__FILE__) + '/id_rsa.pub').chomp
    }
  }
}

puts JSON.pretty_generate(output)

