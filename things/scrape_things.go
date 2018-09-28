package main;


import "io/ioutil"
import "strings"
import "fmt"
import "net/http"
import "errors"
import "regexp"
import "strconv"
import "os"
import "encoding/json"

type JsonComponent struct{
    Name string `json:"name"`
    Num float64 `json:"num"`
}

type JsonThing struct{
    Name string `json:"name"`
    Secs float64 `json:"secs"`
    Prod float64 `json:"prod"`
    Components []JsonComponent `json:"components"`
}

var get_name_url_re *regexp.Regexp=regexp.MustCompile("\\<a href=\\\"\\/([a-zA-Z0-9_-]+)\\\" title=\\\"([a-zA-Z0-9_ -]+)\\\"\\>")
var get_num_re *regexp.Regexp=regexp.MustCompile("\\>([0-9\\.]{1,5}) *\\<\\/div\\>\\<\\/div\\>")
func get_name_url_num(html_line string) (string,string,string){
    name:=""
    url:=""
    num:=""
    found:=get_name_url_re.FindAllStringSubmatch(html_line, -1)
    if len(found)==1{
        name=found[0][2]
        url=found[0][1]
    }

    found=get_num_re.FindAllStringSubmatch(html_line, -1)
    if len(found)==1{
        num=found[0][1]
    }

    return name, url, num
}

type UnusualLayoutError struct{}

func (u UnusualLayoutError) Error() string{
    return "Unusual layout"
}

func scrape(thing_name string) (JsonThing,error) {
    response, err:=http.Get(fmt.Sprintf("https://wiki.factorio.com/%s", thing_name))
    if err!=nil{
        return JsonThing{}, err
    }
    content_bytes,err:=ioutil.ReadAll(response.Body)
    if err!=nil{
        return JsonThing{}, err
    }
    content_string:=string(content_bytes)

    recipe_index:=strings.Index(content_string, "<p>Recipe")
    recipe_total_raw:=strings.Index(content_string, "<p>Total raw")
    if recipe_index==-1 || recipe_total_raw==-1 || recipe_total_raw<=recipe_index{
        return JsonThing{}, UnusualLayoutError{}
    }

    json_thing:=JsonThing{}
    string_to_observe:=content_string[recipe_index:recipe_total_raw]
    for _,line_to_observe:=range strings.Split(string_to_observe, "\n"){
        name, url, num_string:=get_name_url_num(line_to_observe)
        if len(name)==0 || len(url)==0 || len(num_string)==0{
            continue
        }

        num, err:=strconv.ParseFloat(num_string, 64)
        if err!=nil{
            return JsonThing{}, err
        }

        if url=="Time"{
            json_thing.Secs=num
        } else if url==thing_name{
            json_thing.Prod=num
            json_thing.Name=strings.ToLower(strings.Replace(name, " ", "_", -1))
        } else{
            json_thing.Components=append(json_thing.Components, JsonComponent{strings.ToLower(strings.Replace(name, " ", "_", -1)), num})
        }
    }

    if len(json_thing.Name)==0 || json_thing.Secs==0.0 || json_thing.Prod==0.0{
        return JsonThing{}, errors.New("Some field could not be filled")
    }

    return json_thing, nil
}

func main() {
    to_scrape_bytes, err:=ioutil.ReadFile("to_scrape.txt")
    if err!=nil{
        panic("Coudl not open `to_scrape.txt`")
    }

    from_belt_bytes, err:=ioutil.ReadFile("from_belt.txt")
    if err!=nil{
        panic("Coudl not open `from_belt.txt`")
    }

    from_belt_names:=strings.Split(string(from_belt_bytes), "\n")

    json_things:=make([]JsonThing, 0)
    to_scrape_strings:=strings.Split(string(to_scrape_bytes), "\n")
    for _,thing_to_scrape:=range to_scrape_strings{
        json_thing, err:=scrape(thing_to_scrape)
        if err!=nil{
            fmt.Fprintln(os.Stderr, thing_to_scrape, ":", err)
            if _,ok:=err.(UnusualLayoutError); ok{
                json_things=append(json_things, JsonThing{strings.ToLower(strings.Replace(thing_to_scrape, " ", "_", -1)), 1.0, 26.66, make([]JsonComponent, 0)})
            }
        } else {
            fmt.Fprintln(os.Stderr, json_thing)
            for _,from_belt_name:=range from_belt_names{
                if json_thing.Name==from_belt_name{
                    json_thing.Prod=26.66
                    json_thing.Secs=1.0
                    json_thing.Components=make([]JsonComponent, 0)
                    break
                }
            }
            json_things=append(json_things, json_thing)
        }
    }

    json, err:=json.MarshalIndent(json_things, "", "    ")
    if err!=nil{
        panic(err)
    }

    fmt.Println(string(json))
}