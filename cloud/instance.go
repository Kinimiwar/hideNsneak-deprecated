package cloud

//Structs in use throughout the application//
import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/digitalocean/godo"
	"github.com/rmikehodges/hideNsneak/amazon"
	"github.com/rmikehodges/hideNsneak/do"
	"github.com/rmikehodges/hideNsneak/sshext"
)

type Config struct {
	PublicKey   string `yaml:"PublicKey"`
	Customer    string `yaml:"Customer"`
	StartPort   int    `yaml:"StartPort"`
	Proxychains string `yaml:"Proxychains"`
	CSDir       string `yaml:"CobaltStrikeDir"`
	CSProfiles  string `yaml:"CobaltStrikeProfiles"`
	CSLicense   string `yaml:"CobaltStrikeLicense"`
	NmapDir     string `yaml:"NmapDir"`
	AWS         struct {
		Secret         string `yaml:"secret"`
		AccessID       string `yaml:"accessID"`
		Regions        string `yaml:"regions"`
		ImageIDs       string `yaml:"imageIDs"`
		Type           string `yaml:"type"`
		Number         int    `yaml:"number"`
		Termination    map[string][]string
		SecurityGroups map[string][]string
	} `yaml:"AWS"`
	DO struct {
		Token       string `yaml:"token"`
		Fingerprint string `yaml:"fingerprint"`
		Regions     string `yaml:"regions"`
		Slug        string `yaml:"slug"`
		Memory      string `yaml:"memory"`
		Name        string `yaml:"name"`
		Number      int    `yaml:"number"`
	} `yaml:"DO"`
	Google struct {
		ImageFamily  string `yaml:"imageFamily"`
		ImageProject string `yaml:"imageProject"`
		MachineType  string `yaml:"machineType"`
		Zones        string `yaml:"zones"`
		Number       int    `yaml:"number"`
	} `yaml:"Google"`
}

type Instance struct {
	Cloud struct {
		Config      Config
		Type        string
		Description string
		ID          string
		Region      string
		IPv4        string
		Firewalls   []string
		Domain      string
		HomeDir     string
	}
	SSH struct {
		Username   string
		PrivateKey string
	}
	Proxy struct {
		SOCKSActive bool
		SOCKSPort   string
		Process     *os.Process
	}
	SSL struct {
		SSLKeyPass   string
		CertLocation string
		SSLEnabled   bool
	}
	Nmap struct {
		NmapTargets  map[string][]string
		NmapActive   bool
		NmapCmd      string
		NmapProcess  string
		TimeWindow   string
		NmapLocalDir string
	}
	DomainFront struct {
		DomainFrontURL string
		DomainFront    string
	}
	CobaltStrike struct {
		C2Profile         string
		CSPassword        string
		TeamserverEnabled bool
		KillDate          string
	}
}

//String() prints generic information for the user
func (instance Instance) String() string {
	socksPort := ""
	nmapActive := "N"
	if instance.Proxy.SOCKSActive {
		socksPort = instance.Proxy.SOCKSPort
	}
	if instance.Nmap.NmapActive {
		nmapActive := "Y"
	}
	fmt.Printf("Type: %s | IP: %s | Region: %s | Nmap Active: %s | SOCKS: %s", instance.Cloud.Type, instance.Cloud.IPv4,
		instance.Cloud.Region, nmapActive, socksPort)
}

//Detail() prints all information about the instance
func (instance Instance) Detail() string {
	return ""
}

//Start, Stop, Initialize
func StartInstances(config Config, providerMap map[string]int) ([]*Instance, map[string][]string) {
	var cloudInstances []Instance
	var instanceArray []*Instance
	var terminationMap map[string][]string
	var ec2Instances []*ec2.Instance

	for provider, count := range providerMap {
		//Instance Creation
		switch provider {
		case "AWS":
			ec2Instances = amazon.DeployMultipleEC2(config.AWS.Secret, config.AWS.AccessID,
				splitOnComma(config.AWS.Regions), splitOnComma(config.AWS.ImageIDs), count,
				config.PublicKey, config.AWS.Type)
			cloudInstances = append(cloudInstances, ec2ToInstance(ec2Instances, terminationMap)...)
		case "DO":
			doInstances := do.DeployDO(config.DO.Token, config.DO.Regions, config.DO.Memory, config.DO.Slug,
				config.DO.Fingerprint, count, config.DO.Name)
			cloudInstances = append(cloudInstances, dropletsToInstances(doInstances, config)...)
		case "Azure":
			//To be added
		case "Google":
			//To be added
		default:
			continue
		}
	}
	if len(cloudInstances) > 0 {
		fmt.Println("Waiting a few seconds for all instances to initialize...")
		time.Sleep(60 * time.Second)

		for i := range cloudInstances {
			instanceArray = append(instanceArray, &cloudInstances[i])
		}

		getIPAddresses(instanceArray, config)

		//Logging Creation of instances
		for _, instance := range instanceArray {
			WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " created")
		}
	}

	return instanceArray, terminationMap
}

//TODO: Add Stopping of Google and Azure instances
func StopInstances(config Config, allInstances []*Instance) {

	for _, instance := range allInstances {
		switch instance.Cloud.Type {
		case "DO":
			id, _ := strconv.Atoi(instance.Cloud.ID)
			if err := do.DestroyDOInstance(config.DO.Token, instance.Cloud.ID) != nil {
				WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " destroyed")
			} else {
				WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not destroyed - see error log")
				WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + err)
			}
		case "AWS":
			if err := amazon.TerminateInstances(instance.) != nil {
				WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " destroyed")
			} else {
				WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " not destroyed - see error log")
				WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + ":" + err)
			}
		case "Google":
		case "Azure":
		}
	}
	stopSocks(allInstances)
	fmt.Println("About to terminate")
	//
}

//stopSocks loops through a set of instances and kills their SOCKS processes
func stopSOCKS(allInstances []*Instance) {
	for _, instance := range allInstances {
		if instance.Proxy.SOCKSActive == true && instance.Proxy.Process != nil {
			err := instance.Proxy.Process.Kill()
			if err != nil {
				WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " SOCKS not destroyed- see error log")
				WriteErrorLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + "Error killing SOCKS process:" + err)
				continue
			}
			instance.Proxy.SOCKSActive = false
			WriteActivityLog(instance.Cloud.Type + " " + instance.Cloud.IPv4 + " " + instance.Cloud.Region + " : SOCKS destroyed":)
		}
	}
}

func Initialize(allInstances []*Instance, config Config) {
	for _, instance := range allInstances {
		instance.SSH.PrivateKey = strings.Split(config.PublicKey, ".pub")[0]
		instance.Cloud.HomeDir = sshext.SetHomeDir(instance.Cloud.IPv4, instance.SSH.Username, instance.SSH.PrivateKey)
		instance.Proxy.SOCKSActive = false
		instance.CobaltStrike.TeamserverEnabled = false
		instance.Nmap.NmapActive = false
	}
}

//Converting Custom cloud objects to Instance objects
func dropletsToInstances(droplets []godo.Droplet, config Config) []Instance {
	var Instances []Instance
	for _, drop := range droplets {

		IP, err := drop.PublicIPv4()
		if err != nil {
			log.Fatalf("Unable to get ip address for %s", drop)
		}
		tempInstance := Instance{}
		tempInstance.Cloud.Type = "DO"
		tempInstance.Cloud.ID = strconv.Itoa(drop.ID)
		tempInstance.Cloud.Region = drop.Region.Slug
		tempInstance.Cloud.IPv4 = IP

		tempInstance.SSH.Username = "root"
		Instances = append(Instances, tempInstance)
	}
	return Instances
}

func ec2ToInstance(runResult []*ec2.Instance, regionMap map[string][]string) (ec2Instances []Instance) {
	var ec2Instance Instance
	for region, instanceIDs := range regionMap {
		for _, instance := range runResult {
			if contains(instanceIDs, *instance.InstanceId) {
				ec2Instance.Cloud.ID = aws.StringValue(instance.InstanceId)
				ec2Instance.Cloud.IPv4 = aws.StringValue(instance.PublicIpAddress)
				ec2Instance.Cloud.Type = "EC2"
				ec2Instance.SSH.Username = "ubuntu"
				ec2Instance.Cloud.Region = region
				ec2Instances = append(ec2Instances, ec2Instance)
			}
		}
	}
	return
}

func (config *Config) updateTermination(terminationMap map[string][]string) {
	config.AWS.Termination = terminationMap
}

func getIPAddresses(allInstances []*Instance, config Config) {
	for _, instance := range allInstances {
		if instance.Cloud.Type == "EC2" {
			instance.Cloud.IPv4 = amazon.GetEC2IP(instance.Cloud.Region, config.AWS.Secret, config.AWS.AccessID, instance.Cloud.ID)
		}
		if instance.Cloud.Type == "DO" {
			doID, _ := strconv.Atoi(instance.Cloud.ID)
			instance.Cloud.IPv4 = do.GetDOIP(config.DO.Token, doID)
		}
	}
	// return allInstances
}

func DestroyAllDroplets(token string) {
	do.DestroyAllDrops(token)
}
