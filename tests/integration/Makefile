.PHONEY: all ssh_key sshnat go integration

all: ssh_key .terraform variables.tf.json
		true

go:
	go get -a -x -installsuffix cgo -ldflags '-d -s -w' github.com/onsi/ginkgo/ginkgo
	go get -a -x -installsuffix cgo -ldflags '-d -s -w' github.com/onsi/gomega

.terraform_applied:
	terraform apply
	sleep 35
	touch .terraform_applied

integration: go .terraform_applied
	ginkgo -v

sshnat:
	ssh -A -i id_rsa ubuntu@$$(terraform output nat_public_ips|cut -d, -f1)

.terraform:
	terraform get
	for i in $$(ls .terraform/modules/); do make -C ".terraform/modules/$$i"; done

ssh_key: id_rsa id_rsa.pub

id_rsa:
		ssh-keygen -t rsa -f id_rsa -N ''

id_rsa.pub:
		ssh-keygen -y -f id_rsa > id_rsa.pub

variables.tf.json:
		ruby getvariables.rb > variables.tf.json

clean:
		rm -rf .terraform .terraform_applied id_rsa id_rsa.pub variables.tf.json

destroy:
	yes yes | terraform destroy

realclean: .terraform destroy clean

