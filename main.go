package main

import (
	"fmt"
	"strings"
	"bufio"
	"os"
	"net/http"
	"encoding/json"
	"io"
	"log"
	"pokedexcli/internal"
	"time"
	"strconv"
	"math"
	"math/rand"
)

var MAP_INDEX int
var MAP_CACHE internal.Cache
var EXPLORE_CACHE internal.Cache
var CATCH_CACHE internal.Cache
var POKEMON map[string]pokemonEntry
var CAUGHT map[string]struct{}

type cliCommand struct {
	name string
	description string
	callback func(location string) error
}

type locationArea struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

type locationAreaLocation struct {
	EncounterMethodRates []struct {
		EncounterMethod struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"encounter_method"`
		VersionDetails []struct {
			Rate    int `json:"rate"`
			Version struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"version"`
		} `json:"version_details"`
	} `json:"encounter_method_rates"`
	GameIndex int `json:"game_index"`
	ID        int `json:"id"`
	Location  struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"location"`
	Name  string `json:"name"`
	Names []struct {
		Language struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"language"`
		Name string `json:"name"`
	} `json:"names"`
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"pokemon"`
		VersionDetails []struct {
			EncounterDetails []struct {
				Chance          int   `json:"chance"`
				ConditionValues []any `json:"condition_values"`
				MaxLevel        int   `json:"max_level"`
				Method          struct {
					Name string `json:"name"`
					URL  string `json:"url"`
				} `json:"method"`
				MinLevel int `json:"min_level"`
			} `json:"encounter_details"`
			MaxChance int `json:"max_chance"`
			Version   struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"version"`
		} `json:"version_details"`
	} `json:"pokemon_encounters"`
}

func createRegistry() map[string]cliCommand{
	return map[string]cliCommand {
		"exit": {
			name: "exit",
			description:"Exit the Pokedex",
			callback: commandExit,
		},
		"help": {
			name: "help",
			description: "List commands",
			callback: help,
		},
		"map": {
			name:"map",
			description: "Displays the next 20 location names.",
			callback: pokeMap,
		},
		"mapb": {
			name:"mapb",
			description: "Displays the previous 20 location names.",
			callback: pokeMapB,
		},
		"explore": {
			name:"explore",
			description:"Show pokemon at location, type explore <location>",
			callback:exploreMap,
		},
		"catch": {
			name:"catch",
			description:"Throw a pokeball at a pokemon, type catch <pokemon name>",
			callback:catch,
		},
		"inspect": {
			name:"inspect",
			description:"Inspect a pokemon, type inspect <pokemon name>",
			callback:inspect,
		},
		"pokedex": {
			name:"pokedex",
			description:"List caught pokemon names",
			callback:pPokedex,
		},
	}
}

func printMaps(offset int, limit int ) error {
	mapBytes, isCached := MAP_CACHE.Get(strconv.Itoa(offset))
	if (isCached) {
		fmt.Println(string(mapBytes))
		return nil
	}


	query := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/?offset=%v&limit=%v",offset,limit)
	resp,httpErr := http.Get(query)
	if (httpErr != nil) {
		return httpErr
	}
	body, readErr := io.ReadAll(resp.Body)
	if (readErr != nil)  {
		return readErr
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		log.Fatalf("Failed %d and %s",resp.StatusCode, body)
	}
	var la locationArea
	jErr := json.Unmarshal(body, &la)
	if (jErr != nil) {
		return jErr
	}
	var mapList string
	for _,obj := range la.Results {
		mapList = mapList + fmt.Sprintf("%s\n",obj.Name)
	}
	MAP_CACHE.Add(strconv.Itoa(offset), []byte(mapList))
	fmt.Println(mapList)
	return nil
}

func pokeMap(location string) error {
	MAP_INDEX++
	limit := 20
	offset := 20 * MAP_INDEX
	err := printMaps(offset, limit)
	return err
}

func pokeMapB(location string) error {
	limit := 20
	mapIndex := MAP_INDEX-1
	if (MAP_INDEX-1 < 0) {
		mapIndex = 0
		if (MAP_INDEX-1 == 0) {
			MAP_INDEX--
		}
	} else {
		MAP_INDEX--
	}
	offset := 20 * mapIndex
	err := printMaps(offset, limit)
	return err

}

func printPokemon(location string) error {
	monBytes, isCached := EXPLORE_CACHE.Get(location)
	if (isCached) {
		fmt.Println(string(monBytes))
		return nil
	}


	query := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s",location)
	resp,httpErr := http.Get(query)
	if (httpErr != nil) {
		return httpErr
	}
	body, readErr := io.ReadAll(resp.Body)
	if (readErr != nil)  {
		return readErr
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		log.Fatalf("Failed %d and %s",resp.StatusCode, body)
	}
	var lal locationAreaLocation
	jErr := json.Unmarshal(body, &lal)
	if (jErr != nil) {
		return jErr
	}
	var monList string
	monList += fmt.Sprintf("Exploring %s...\n", location)
	monList += fmt.Sprintf("Found Pokemon:\n")
	for _,obj := range lal.PokemonEncounters {
		monList = monList + fmt.Sprintf(" - %s\n",obj.Pokemon.Name)
	}
	EXPLORE_CACHE.Add(location, []byte(monList))
	fmt.Println(monList)
	return nil
}

func exploreMap(location string) error {
	return printPokemon(location)
}

func catch(name string) error {
	fmt.Printf("Throwing a Pokeball at %s...\n",name)
	monBytes, isCached := EXPLORE_CACHE.Get(name)
	if (isCached) {
		if monBytes != nil {
			bexp,_ := strconv.Atoi(string(monBytes))
			chance := (1/(math.Log(float64(bexp))))*100
			isCaught := roll(chance)
			if isCaught {
				CAUGHT[name]=struct{}{}
				fmt.Printf("%s was caught!\n",name)
			} else {
				fmt.Printf("%s escaped!\n", name)
			}
		} else {
			fmt.Printf("Pokemon %s not found in pokedex\n", name)
		}
		return nil
	}

	query := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s",name)
	resp,httpErr := http.Get(query)
	if (httpErr != nil) {
		return httpErr
	}
	if resp.StatusCode == 404 {
		fmt.Printf("Pokemon %s not found in pokedex\n", name)
		EXPLORE_CACHE.Add(name, nil)
		return nil
	}
	body, readErr := io.ReadAll(resp.Body)
	if (readErr != nil)  {
		return readErr
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		log.Fatalf("Failed %d and %s",resp.StatusCode, body)
	}

	var mon pokemonEntry
	jErr := json.Unmarshal(body, &mon)
	if (jErr != nil) {
		return jErr
	}

	POKEMON[name]=mon
	EXPLORE_CACHE.Add(name, []byte(strconv.Itoa(mon.BaseExperience)))	
	
	chance := (1/(math.Log(float64(mon.BaseExperience))))*100
	isCaught := roll(chance)
	if isCaught {
		CAUGHT[name]=struct{}{}
		fmt.Printf("%s was caught!\n",name)
	} else {
		fmt.Printf("%s escaped!\n", name)
	}
	return nil
}

func roll(pct float64) bool {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(100) < int(pct)
}

func help(location string) error {
	cmdMap := createRegistry()
	fmt.Println("Commands")
	fmt.Println("name: description")
	
	for _, cmdObj := range cmdMap {
		fmt.Printf("%v:\t%v\n", cmdObj.name, cmdObj.description)
	}
	return nil
}

func inspect(name string) error {
	mon,exists := POKEMON[name]
	if  exists {
		fmt.Printf("Name: %s\n", name)
		fmt.Printf("height: %v\n", mon.Height)
		fmt.Printf("weight: %v\n", mon.Weight)
		fmt.Printf("stats:\n")
		for _,stat := range mon.Stats {
			if stat.Stat.Name == "hp" || stat.Stat.Name == "attack" || stat.Stat.Name == "defense" || stat.Stat.Name == "special-attack" || stat.Stat.Name == "special-defense" || stat.Stat.Name == "speed" {
				fmt.Printf("  -%s: %v\n",stat.Stat.Name, stat.BaseStat)
			}
		}
		fmt.Printf("types:\n")
		for _,pType := range mon.Types {
			fmt.Printf("  - %s\n",pType.Type.Name)
		}
	} else {
		fmt.Printf("Unknown pokemon, try catching one with catch %s\n", name)
	}
	return nil
}

func pPokedex(unused string) error {
	fmt.Println("Your Pokedex:")
	for key,_ := range CAUGHT {
		fmt.Printf(" - %s\n",key)
	}
	return nil
} 

func main() {
	MAP_INDEX = -1 // Redundant, but do note.
	MAP_CACHE = internal.NewCache(time.Second*5)
	EXPLORE_CACHE = internal.NewCache(time.Second*5)
	CATCH_CACHE = internal.NewCache(time.Second*5)
	POKEMON=make(map[string]pokemonEntry)
	CAUGHT=make(map[string]struct{})
	fmt.Println("Welcome to the Pokedex!")
	scanner := bufio.NewScanner(os.Stdin)
	cmdMap := createRegistry()
	for {
		fmt.Print("Pokedex > ")
		if scanner.Scan() {
			text :=scanner.Text()
			userCmd := cleanInput(text)
			if len(userCmd) < 1 {
				fmt.Println("Unknown command")
			} else {
				fmt.Printf("Your command was: %s\n", userCmd[0])
				cmdObj, cmdExists := cmdMap[userCmd[0]]
				if cmdExists {
					if (len(userCmd) > 1) {
						cmdObj.callback(userCmd[1])
					} else {
						cmdObj.callback("")
					}
				} else {
					fmt.Println("Unknown command")
				}
			}
		} else {
			fmt.Println("ERROR READING CLI INPUT")
		}
	}
}

func commandExit(location string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func cleanInput(text string) []string {
	words := strings.Fields(text)
	for i:=range words {
		words[i]=strings.ToLower(words[i])
	}
	return words
}

type pokemonEntry struct {
	Abilities []struct {
		Ability struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"ability"`
		IsHidden bool `json:"is_hidden"`
		Slot     int  `json:"slot"`
	} `json:"abilities"`
	BaseExperience int `json:"base_experience"`
	Cries          struct {
		Latest string `json:"latest"`
		Legacy string `json:"legacy"`
	} `json:"cries"`
	Forms []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"forms"`
	GameIndices []struct {
		GameIndex int `json:"game_index"`
		Version   struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"version"`
	} `json:"game_indices"`
	Height                 int    `json:"height"`
	HeldItems              []any  `json:"held_items"`
	ID                     int    `json:"id"`
	IsDefault              bool   `json:"is_default"`
	LocationAreaEncounters string `json:"location_area_encounters"`
	Moves                  []struct {
		Move struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"move"`
		VersionGroupDetails []struct {
			LevelLearnedAt  int `json:"level_learned_at"`
			MoveLearnMethod struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"move_learn_method"`
			Order        any `json:"order"`
			VersionGroup struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"version_group"`
		} `json:"version_group_details"`
	} `json:"moves"`
	Name          string `json:"name"`
	Order         int    `json:"order"`
	PastAbilities []struct {
		Abilities []struct {
			Ability  any  `json:"ability"`
			IsHidden bool `json:"is_hidden"`
			Slot     int  `json:"slot"`
		} `json:"abilities"`
		Generation struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"generation"`
	} `json:"past_abilities"`
	PastTypes []any `json:"past_types"`
	Species   struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"species"`
	Sprites struct {
		BackDefault      string `json:"back_default"`
		BackFemale       any    `json:"back_female"`
		BackShiny        string `json:"back_shiny"`
		BackShinyFemale  any    `json:"back_shiny_female"`
		FrontDefault     string `json:"front_default"`
		FrontFemale      any    `json:"front_female"`
		FrontShiny       string `json:"front_shiny"`
		FrontShinyFemale any    `json:"front_shiny_female"`
		Other            struct {
			DreamWorld struct {
				FrontDefault string `json:"front_default"`
				FrontFemale  any    `json:"front_female"`
			} `json:"dream_world"`
			Home struct {
				FrontDefault     string `json:"front_default"`
				FrontFemale      any    `json:"front_female"`
				FrontShiny       string `json:"front_shiny"`
				FrontShinyFemale any    `json:"front_shiny_female"`
			} `json:"home"`
			OfficialArtwork struct {
				FrontDefault string `json:"front_default"`
				FrontShiny   string `json:"front_shiny"`
			} `json:"official-artwork"`
			Showdown struct {
				BackDefault      string `json:"back_default"`
				BackFemale       any    `json:"back_female"`
				BackShiny        string `json:"back_shiny"`
				BackShinyFemale  any    `json:"back_shiny_female"`
				FrontDefault     string `json:"front_default"`
				FrontFemale      any    `json:"front_female"`
				FrontShiny       string `json:"front_shiny"`
				FrontShinyFemale any    `json:"front_shiny_female"`
			} `json:"showdown"`
		} `json:"other"`
		Versions struct {
			GenerationI struct {
				RedBlue struct {
					BackDefault      string `json:"back_default"`
					BackGray         string `json:"back_gray"`
					BackTransparent  string `json:"back_transparent"`
					FrontDefault     string `json:"front_default"`
					FrontGray        string `json:"front_gray"`
					FrontTransparent string `json:"front_transparent"`
				} `json:"red-blue"`
				Yellow struct {
					BackDefault      string `json:"back_default"`
					BackGray         string `json:"back_gray"`
					BackTransparent  string `json:"back_transparent"`
					FrontDefault     string `json:"front_default"`
					FrontGray        string `json:"front_gray"`
					FrontTransparent string `json:"front_transparent"`
				} `json:"yellow"`
			} `json:"generation-i"`
			GenerationIi struct {
				Crystal struct {
					BackDefault           string `json:"back_default"`
					BackShiny             string `json:"back_shiny"`
					BackShinyTransparent  string `json:"back_shiny_transparent"`
					BackTransparent       string `json:"back_transparent"`
					FrontDefault          string `json:"front_default"`
					FrontShiny            string `json:"front_shiny"`
					FrontShinyTransparent string `json:"front_shiny_transparent"`
					FrontTransparent      string `json:"front_transparent"`
				} `json:"crystal"`
				Gold struct {
					BackDefault      string `json:"back_default"`
					BackShiny        string `json:"back_shiny"`
					FrontDefault     string `json:"front_default"`
					FrontShiny       string `json:"front_shiny"`
					FrontTransparent string `json:"front_transparent"`
				} `json:"gold"`
				Silver struct {
					BackDefault      string `json:"back_default"`
					BackShiny        string `json:"back_shiny"`
					FrontDefault     string `json:"front_default"`
					FrontShiny       string `json:"front_shiny"`
					FrontTransparent string `json:"front_transparent"`
				} `json:"silver"`
			} `json:"generation-ii"`
			GenerationIii struct {
				Emerald struct {
					FrontDefault string `json:"front_default"`
					FrontShiny   string `json:"front_shiny"`
				} `json:"emerald"`
				FireredLeafgreen struct {
					BackDefault  string `json:"back_default"`
					BackShiny    string `json:"back_shiny"`
					FrontDefault string `json:"front_default"`
					FrontShiny   string `json:"front_shiny"`
				} `json:"firered-leafgreen"`
				RubySapphire struct {
					BackDefault  string `json:"back_default"`
					BackShiny    string `json:"back_shiny"`
					FrontDefault string `json:"front_default"`
					FrontShiny   string `json:"front_shiny"`
				} `json:"ruby-sapphire"`
			} `json:"generation-iii"`
			GenerationIv struct {
				DiamondPearl struct {
					BackDefault      string `json:"back_default"`
					BackFemale       any    `json:"back_female"`
					BackShiny        string `json:"back_shiny"`
					BackShinyFemale  any    `json:"back_shiny_female"`
					FrontDefault     string `json:"front_default"`
					FrontFemale      any    `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale any    `json:"front_shiny_female"`
				} `json:"diamond-pearl"`
				HeartgoldSoulsilver struct {
					BackDefault      string `json:"back_default"`
					BackFemale       any    `json:"back_female"`
					BackShiny        string `json:"back_shiny"`
					BackShinyFemale  any    `json:"back_shiny_female"`
					FrontDefault     string `json:"front_default"`
					FrontFemale      any    `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale any    `json:"front_shiny_female"`
				} `json:"heartgold-soulsilver"`
				Platinum struct {
					BackDefault      string `json:"back_default"`
					BackFemale       any    `json:"back_female"`
					BackShiny        string `json:"back_shiny"`
					BackShinyFemale  any    `json:"back_shiny_female"`
					FrontDefault     string `json:"front_default"`
					FrontFemale      any    `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale any    `json:"front_shiny_female"`
				} `json:"platinum"`
			} `json:"generation-iv"`
			GenerationV struct {
				BlackWhite struct {
					Animated struct {
						BackDefault      string `json:"back_default"`
						BackFemale       any    `json:"back_female"`
						BackShiny        string `json:"back_shiny"`
						BackShinyFemale  any    `json:"back_shiny_female"`
						FrontDefault     string `json:"front_default"`
						FrontFemale      any    `json:"front_female"`
						FrontShiny       string `json:"front_shiny"`
						FrontShinyFemale any    `json:"front_shiny_female"`
					} `json:"animated"`
					BackDefault      string `json:"back_default"`
					BackFemale       any    `json:"back_female"`
					BackShiny        string `json:"back_shiny"`
					BackShinyFemale  any    `json:"back_shiny_female"`
					FrontDefault     string `json:"front_default"`
					FrontFemale      any    `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale any    `json:"front_shiny_female"`
				} `json:"black-white"`
			} `json:"generation-v"`
			GenerationVi struct {
				OmegarubyAlphasapphire struct {
					FrontDefault     string `json:"front_default"`
					FrontFemale      any    `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale any    `json:"front_shiny_female"`
				} `json:"omegaruby-alphasapphire"`
				XY struct {
					FrontDefault     string `json:"front_default"`
					FrontFemale      any    `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale any    `json:"front_shiny_female"`
				} `json:"x-y"`
			} `json:"generation-vi"`
			GenerationVii struct {
				Icons struct {
					FrontDefault string `json:"front_default"`
					FrontFemale  any    `json:"front_female"`
				} `json:"icons"`
				UltraSunUltraMoon struct {
					FrontDefault     string `json:"front_default"`
					FrontFemale      any    `json:"front_female"`
					FrontShiny       string `json:"front_shiny"`
					FrontShinyFemale any    `json:"front_shiny_female"`
				} `json:"ultra-sun-ultra-moon"`
			} `json:"generation-vii"`
			GenerationViii struct {
				Icons struct {
					FrontDefault string `json:"front_default"`
					FrontFemale  any    `json:"front_female"`
				} `json:"icons"`
			} `json:"generation-viii"`
		} `json:"versions"`
	} `json:"sprites"`
	Stats []struct {
		BaseStat int `json:"base_stat"`
		Effort   int `json:"effort"`
		Stat     struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Slot int `json:"slot"`
		Type struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"type"`
	} `json:"types"`
	Weight int `json:"weight"`
}