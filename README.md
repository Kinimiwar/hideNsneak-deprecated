# hideNsneak

## Disclosure: This tool is still in beta so use at your own risk. Full version will be released around Blackhat this year


This application assists in managing attack infrasturcture by providing an interface to rapidly deploy, manage, and take down various cloud services. These include VMs, domain fronting, Cobalt Strike servers, API gateways, and firewalls. 

#running locally

`git clone https://github.com/rmikehodges/hideNsneak.git`
`cd hideNsneak/main`
`go get github.com/rmikehodges/hideNsneak/cloud`
`go get github.com/rmikehodges/hideNsneak/misc`
`go get github.com/rmikehodges/hideNsneak/sshext`
`go run main.go`

fill in the values in config.yaml with API keys, file paths, etc (see below for more info)

set up your ssh key in your config.yaml file with all cloud provider you'd like to use (AWS, Google, Digital Ocean)
