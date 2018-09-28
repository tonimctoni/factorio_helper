extern crate serde_json;
#[macro_use]
extern crate serde_derive;
use std::fs;
use std::collections::HashMap;
use std::env::args;

#[derive(Debug)]
struct Thing {
    secs: f64,
    prod: f64,
    components: Vec<(String, f64)>
}

#[derive(Debug)]
struct Things(HashMap<String, Thing>);

impl Things {
    fn new(json_filename: &str) -> Self{
        #[derive(Deserialize)]
        struct JsonComponent {
            name: String,
            num: f64
        }

        #[derive(Deserialize)]
        struct JsonThing {
            name: String,
            secs: f64,
            prod: f64,
            components: Vec<JsonComponent>
        }

        let json_file=fs::File::open(json_filename).expect("Could not open json file");
        let things: Vec<JsonThing>=serde_json::from_reader(json_file).expect("Could not parse json file");

        Things(things.into_iter().map(|JsonThing{name, secs, prod, components}|{
            (
                name,
                Thing{
                    prod: prod,
                    secs: secs,
                    components: components.into_iter().map(|JsonComponent{name, num}| {
                        (name, num)
                    }).collect()
                }
            )
        }).collect())
    }


    fn add_needed(&self, name: &String, per_second: f64, needed_assemblies: &mut HashMap<String, f64>){
        let thing=self.0.get(name).expect(&format!("Could not find `{}` in map", name));
        let needed_assemblies_for_name=(per_second*thing.secs)/thing.prod; //number of assemblies needed
        needed_assemblies
            .entry(name.clone())
            .and_modify(|assemblies| *assemblies+=needed_assemblies_for_name)
            .or_insert(needed_assemblies_for_name);

        for (name, times) in thing.components.iter(){
            self.add_needed(name, (times*per_second)/thing.prod, needed_assemblies);
        }
    }
}

fn main() {
    let mut args=args().skip(1);
    let thing=args.next().expect("Two arguments are needed");
    let per_second=args.next().and_then(|s| s.parse::<f64>().ok()).expect("Two arguments are needed; the second one has to be a number");
    drop(args);

    let things=Things::new("things.json");

    let mut needed_assemblies: HashMap<String, f64>=HashMap::new();
    things.add_needed(&thing, per_second, &mut needed_assemblies);
    let mut needed_assemblies=needed_assemblies.into_iter().collect::<Vec<(String, f64)>>();
    needed_assemblies.sort_by_key(|(_,v)| v.ceil() as isize);
    for (k,v) in needed_assemblies.into_iter(){
        println!("{:20} -> {:3} ({:.4})", k, v.ceil(), v);
    }

    // println!("{:?}", needed_assemblies);
}
