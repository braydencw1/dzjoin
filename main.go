package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/joho/godotenv"
)

type CLI struct {
	Update   bool `short:"u" help:"Update mods."`
	Clean    bool `short:"c" help:"Delete mods in Dayz Folder"`
	DontJoin bool `short:"d" help:"Don't join the server"`
}

func main() {
	cli := CLI{}
	kong.Parse(&cli)
	dzjoinPath := initDirectory()

	err := godotenv.Load(filepath.Join(dzjoinPath, ".env"))
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}

	server := envServerID()
	resp, err := FetchServer(server)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}
	if cli.Clean {
		if err := DeleteAtMods(dayzPath); err != nil {
			log.Fatalf("failed deleting old mods: %s", err)
		}
	}

	if cli.Update {
		err := HandleWorkshop(resp)
		if err != nil {
			log.Fatal(err)
		}
	}
	if !cli.DontJoin {
		err = LaunchGame(resp.Data.Attributes.IP, strconv.Itoa(resp.Data.Attributes.Port), resp.Data.Attributes.Details.ModNames)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func BuildModMap(resp *ServerResponse) map[int64]string {
	ids := resp.Data.Attributes.Details.ModIDs
	names := resp.Data.Attributes.Details.ModNames

	modMap := make(map[int64]string, len(ids))

	for i := range ids {
		modMap[ids[i]] = names[i]
	}

	return modMap
}

func LaunchGame(ip, port string, modNames []string) error {
	for i := range modNames {
		modNames[i] = "@" + modNames[i]
	}

	un := getUserName()

	modArg := "-mod=" + strings.Join(modNames, ";")

	cmd := exec.Command(
		"steam", "-applaunch", "221100", "-noLauncher", "-nosplash", "-skipintro", "-name="+un,
		"-connect="+ip,
		"-port="+port,
		modArg,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func GetSteamCmdPath() (string, error) {
	steamCmdPath := os.Getenv("DZJOIN_STEAMCMD_PATH")

	if steamCmdPath != "" {
		if _, err := os.Stat(steamCmdPath); err != nil {
			return "", fmt.Errorf("steamcmd not found or installed")
		}
		return steamCmdPath, nil
	}

	steamCmdPath, err := exec.LookPath("steamcmd")

	if err != nil {
		return "", fmt.Errorf("steamcmd not found or installed: %w", err)
	}

	return steamCmdPath, nil
}

func envServerID() string {
	server := os.Getenv("DZJOIN_SERVER")
	if server == "" {
		panic("DZJOIN_SERVER not defined")
	}

	return server
}

func FetchServer(server string) (*ServerResponse, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.battlemetrics.com/servers/%s", server))
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatalf("http request failed: %v", err)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	DzInfo := &ServerResponse{}

	err = json.Unmarshal(body, DzInfo)

	if err != nil {
		return nil, err
	}

	return DzInfo, nil
}

type ServerResponse struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			ID         string    `json:"id"`
			Name       string    `json:"name"`
			Address    *string   `json:"address"`
			IP         string    `json:"ip"`
			Port       int       `json:"port"`
			Players    int       `json:"players"`
			MaxPlayers int       `json:"maxPlayers"`
			Rank       int       `json:"rank"`
			Location   []float64 `json:"location"`
			Status     string    `json:"status"`
			Details    struct {
				Version       string   `json:"version"`
				Password      bool     `json:"password"`
				Official      bool     `json:"official"`
				Time          string   `json:"time"`
				ThirdPerson   bool     `json:"third_person"`
				Modded        bool     `json:"modded"`
				ModIDs        []int64  `json:"modIds"`
				ModNames      []string `json:"modNames"`
				ServerSteamID string   `json:"serverSteamId"`
			} `json:"details"`
			Private     bool   `json:"private"`
			CreatedAt   string `json:"createdAt"`
			UpdatedAt   string `json:"updatedAt"`
			PortQuery   int    `json:"portQuery"`
			Country     string `json:"country"`
			QueryStatus string `json:"queryStatus"`
		} `json:"attributes"`
		Relationships struct {
			Game struct {
				Data struct {
					Type string `json:"type"`
					ID   string `json:"id"`
				} `json:"data"`
			} `json:"game"`
		} `json:"relationships"`
	} `json:"data"`
	Included []any `json:"included"`
}

func getHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return home
}

var dayzPath = createDayzPath()

func createDayzPath() string {
	path, err := FindSteamLibrary()
	if err != nil {
		log.Fatalf("err")
	}
	return filepath.Join(path, "common/DayZ/")
}

func HandleWorkshop(resp *ServerResponse) error {

	steamCmdPath, err := GetSteamCmdPath()

	if err != nil {
		return err
	}

	modIDs := resp.Data.Attributes.Details.ModIDs
	if err := DownloadWorkshopMods(steamCmdPath, modIDs); err != nil {
		return fmt.Errorf("failed downloading mod %s", err)
	}
	modMap := BuildModMap(resp)

	if err := MoveWorkshopMod(modMap, dayzPath); err != nil {
		return fmt.Errorf("failed moving mods: %w", err)
	}
	return nil
}

func FindSteamLibrary() (string, error) {
	home := getHome()

	candidates := []string{
		home + "/.steam/steam/steamapps",                                        // default
		home + "/.local/share/Steam/steamapps",                                  // older layout / some distros
		home + "/.var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps", // Flatpak Steam
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("steam library not found")
}

func DownloadWorkshopMods(steamPath string, modids []int64) error {
	args := []string{"+login", "anonymous"}

	for _, m := range modids {
		mod := strconv.FormatInt(m, 10)
		args = append(args, "+workshop_download_item", "221100", mod)
	}
	args = append(args, "+quit")

	cmd := exec.Command(steamPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func MoveWorkshopMod(modMap map[int64]string, dayzPath string) error {
	steamLibrary, err := FindSteamLibrary()

	if err != nil {
		return err
	}

	for modID, modName := range modMap {
		modIDString := strconv.FormatInt(modID, 10)

		workshopPath := filepath.Join(
			steamLibrary,
			"/workshop/content/221100",
			modIDString,
		)

		dest := filepath.Join(dayzPath, "@"+modName)

		if err := os.RemoveAll(dest); err != nil {
			return fmt.Errorf("removing existing mod %s: %w", modName, err)
		}

		if err := CopyDir(workshopPath, dest); err != nil {
			return fmt.Errorf("copying mod %s: %w", modName, err)
		}
	}

	return nil

}

func ParseModName(meta string) string {
	re := regexp.MustCompile(`name\s*=\s*"([^"]+)"`)
	match := re.FindStringSubmatch(meta)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func DeleteAtMods(dayzPath string) error {
	entries, err := os.ReadDir(dayzPath)
	log.Printf("%s", dayzPath)
	if err != nil {
		log.Printf("here")
		return err
	}

	for _, e := range entries {
		if e.IsDir() && len(e.Name()) > 0 && e.Name()[0] == '@' {
			fullPath := filepath.Join(dayzPath, e.Name())
			fmt.Println("Deleting:", fullPath)
			if err := os.RemoveAll(fullPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			err := in.Close()
			if err != nil {
				log.Fatal(err)
			}
		}()

		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer func() {
			err := out.Close()
			if err != nil {
				log.Fatal(err)
			}
		}()

		_, err = io.Copy(out, in)
		return err
	})
}

func getUserName() string {
	name := os.Getenv("DZJOIN_NAME")

	if name == "" {
		log.Fatalf("DZJOIN_NAME not defined.")
	}

	return name
}

func initDirectory() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("cannot get config dir: %v", err)
	}

	path := filepath.Join(dir, "dzjoin")

	if err := os.MkdirAll(path, 0700); err != nil {
		log.Fatalf("cannot create %s: %v", path, err)
	}

	return path
}
