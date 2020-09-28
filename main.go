//go:generate go-bindata -o tpl.go tmpl

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"

	flag "github.com/spf13/pflag"
	"github.com/zorkian/go-datadog-api"
)

type LocalConfig struct {
	client     datadog.Client
	items      []Item
	files      bool
	components []DatadogElement
}

var config = LocalConfig{
	components: []DatadogElement{Monitor{}},
}

type DatadogElement interface {
	getElementById(client datadog.Client, i int) (interface{}, error)
	getElementByTags(client datadog.Client, t []string) ([]Item, error)
	deleteElement(client datadog.Client, i int) error
	getAsset() string
	getName() string
	getAllElements(client datadog.Client) ([]Item, error)
}

type Item struct {
	id int
	d  DatadogElement
}

func (i *Item) getElement(config LocalConfig) (interface{}, error) {
	item, err := i.d.getElementById(config.client, i.id)
	if err != nil {
		log.Debugf("Error while getting element %v", i.id)
		log.Fatal(err)
	}
	return item, err

}

func (i *Item) renderElement(item interface{}, config LocalConfig) {
	log.Debugf("Entering renderElement %v", i.id)
	b, _ := Asset(i.d.getAsset())
	t, _ := template.New("").Funcs(template.FuncMap{
		"escapeCharacters": escapeCharacters,
		"DeRefString":      func(s *string) string { return *s },
	}).Parse(string(b))

	if config.files {
		log.Debug("Creating file", i.d.getName(), i.id)
		file := fmt.Sprintf("%v-%v.tf", i.d.getName(), i.id)
		f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			log.Fatal(err)
		}
		out := bufio.NewWriter(f)
		t.Execute(out, item)
		out.Flush()
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	} else {
		t.Execute(os.Stdout, item)
	}
}

// Replace escaped quote with apostrophe
func escapeCharacters(line string) string {
	return strconv.Quote(line)
}

type SecondaryOptions struct {
	action string
	ids    []int
	mtype  string
	tags   []string
	force  bool
	files  bool
	all    bool
	debug  bool
}

func NewSecondaryOptions(cmd *flag.FlagSet) *SecondaryOptions {
	options := &SecondaryOptions{}
	cmd.StringVar(&options.action, "action", "", "What to do")
	cmd.StringVar(&options.mtype, "type", "", "Monitor type")
	cmd.IntSliceVar(&options.ids, "ids", []int{}, "IDs of the elements to fetch.")
	cmd.StringSliceVar(&options.tags, "tags", []string{}, "Tags of the elements to fetch.")
	cmd.BoolVar(&options.force, "force", false, "Dry run")
	cmd.BoolVar(&options.all, "all", false, "Export all available elements.")
	cmd.BoolVar(&options.files, "files", false, "Save each element into a separate file.")
	cmd.BoolVar(&options.debug, "debug", false, "Enable debug output.")
	return options
}

func executeLogic(opts *SecondaryOptions, config *LocalConfig, component DatadogElement) {
	config.files = opts.files //TODO: get rid of this ugly hack
	if (len(opts.ids) == 0) && (opts.all == false) && (len(opts.tags) == 0) {
		log.Fatal("Either --ids or --all or --tags should be specified")
	} else if opts.all == true {
		allElements, err := component.getAllElements(config.client)
		if err != nil {
			log.Fatal(err)
		}
		config.items = allElements
		log.Debugf("Exporting all elements: %v", allElements)
	} else if len(opts.ids) > 0 {
		log.Debug("Exporting selected elements")
		for _, item := range opts.ids {
			config.items = append(config.items, Item{id: item, d: component})
		}
	} else if len(opts.tags) > 0 {
		log.Debug(opts.tags)
		allElements, err := component.getElementByTags(config.client, opts.tags)
		if err != nil {
			log.Fatal(err)
		}
		config.items = allElements
		log.Debugf("Exporting all elements: %v", allElements)
	}
}

func usage() {
	fmt.Printf("Usage: %v <subcommand> <subcommand_options>\n", os.Args[0])
	fmt.Printf("\twhere <subcommand> is one of: %+v\n", config.components)
	fmt.Println("Environment variables DATADOG_API_KEY and DATADOG_APP_KEY are required")
}

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
	log.RegisterExitHandler(usage)

	if len(os.Args) < 2 {
		log.Fatal("Not enough arguments to proceed")
	} else {
		//TODO: current approach means that we do selective parsing:
		// * some arguments are parsed via os.Args, others - via pflag.Parse()
		// * setting debug level output is complicated;
		//
		// This should be refactored using google/subcommands or something similar

		selected := os.Args[1]
		for _, comp := range config.components {
			if comp.getName() != selected {
				continue
			}
			subcommand := flag.NewFlagSet(selected, flag.ExitOnError)
			subcommandOpts := NewSecondaryOptions(subcommand)
			subcommand.Parse(os.Args[2:])
			if subcommand.Parsed() {
				datadogAPIKey, ok := os.LookupEnv("DATADOG_API_KEY")
				if !ok {
					log.Fatal("Datadog API key not found, please make sure that DATADOG_API_KEY env variable is set")
				}

				datadogAPPKey, ok := os.LookupEnv("DATADOG_APP_KEY")
				if !ok {
					log.Fatal("Datadog APP key not found, please make sure that DATADOG_APP_KEY env variable is set")
				}

				config = LocalConfig{
					client: *datadog.NewClient(datadogAPIKey, datadogAPPKey),
				}

				if subcommandOpts.debug {
					log.SetLevel(log.DebugLevel)
				}
				executeLogic(subcommandOpts, &config, comp)
			}
			if subcommandOpts.action == "export" {
				for _, element := range config.items {
					log.Debugf("Exporting element %v", element.id)
					fullElem, err := element.getElement(config)
					if err != nil {

					}
					monitor := fullElem.(*datadog.Monitor)
					if !strings.Contains(element.d.getName(), " - Terraform") && (subcommandOpts.mtype == "" || *monitor.Type == subcommandOpts.mtype) {
						element.renderElement(fullElem, config)
					}
				}
			} else if subcommandOpts.action == "delete" {
				if !subcommandOpts.force {
					fmt.Println("Dry run")
				}
				for _, element := range config.items {
					fullElem, err := element.getElement(config)
					if err != nil {

					}
					monitor := fullElem.(*datadog.Monitor)
					if !strings.Contains(element.d.getName(), " - Terraform") && (subcommandOpts.mtype == "" || *monitor.Type == subcommandOpts.mtype) {
						found := true
						tm := make(map[string]bool)
						for _, tag := range subcommandOpts.tags {
							tm[tag] = false
						}
						for _, tag := range monitor.Tags {
							tm[tag] = true
						}
						for _, tag := range tm {
							if !tag {
								found = false
								break
							}
						}
						if found && subcommandOpts.force {
							fmt.Printf("Deleting element %v\n", element.id)
							comp.deleteElement(config.client, element.id)
						} else if found {
							fmt.Printf("Will delete element %v\n", element.id)
						}
					}
				}
				log.Debugf("Deleted %v", len(config.items))
			}
			os.Exit(0)

		}
		log.Fatalf("%q is not valid command.\n", os.Args[1])
	}

}
