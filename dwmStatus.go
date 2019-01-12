package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type DwmIcon struct {
	Name string
	Icon string
}

type DwmConfig struct {
	Sound_Device      string
	Network_Interface string
	Icons             []DwmIcon
}

type DwmColor struct {
	start string
	end   string
}

var (
	config_file    string
	dwm_config     DwmConfig
	icons          map[string]string
	collected_data map[string]string
	cores          = runtime.NumCPU()
	rx_old         = 0
	tx_old         = 0
    keys           = []string{"rx", "tx", "brightness", "volume", "temp", "battery", "cpu", "ram", "calendar", "clock"}
	colors         = []DwmColor{{"\x06", "\x07"}, {"\x08", "\x09"}, {"\x0a", "\x0b"}, {"\x0c", "\x0d"}, {"\x0e", "\x0f"}, {"\x10", "\x11"}}
//	colors             = []DwmColor{{"\x0a", "\x0b"}, {"\x0b", "\x0a"}}
	valid_net_device   = false
	valid_sound_device = false
)

func is_valid_net_device() bool {
	out, err := exec.Command("ip", "addr", "show", dwm_config.Network_Interface).Output()
	if err != nil || len(out) == 0 {
		remove_key("rx")
		remove_key("tx")
		fmt.Printf("Network device '%s' is not valid; please recheck the config file\n", dwm_config.Network_Interface)
		return false
	}

	return true
}

func is_valid_sound_device() bool {
	out, err := exec.Command("pactl", "list", "sinks", "short").Output()
	if err != nil || len(out) == 0 {
		fmt.Printf("Sound device '%s' is not valid; please recheck the config file\n", dwm_config.Sound_Device)
		remove_key("volume")
		return false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		device_id := strings.Split(scanner.Text(), "\t")[0]
		if device_id == dwm_config.Sound_Device {
			return true
		}
	}
	remove_key("volume")
	fmt.Printf("Sound device '%s' is not valid; please recheck the config file\n", dwm_config.Sound_Device)
	return false
}

func load_config() {
	json_config, err := os.Open(config_file)

	if err != nil {
		panic(err)
	}
	defer json_config.Close()

	dwm_config = DwmConfig{}
	json_parser := json.NewDecoder(json_config)

	if err = json_parser.Decode(&dwm_config); err != nil {
		panic(err)
	}

	icons = make(map[string]string)

	for i := range dwm_config.Icons {
		icons[dwm_config.Icons[i].Name] = dwm_config.Icons[i].Icon
	}

	valid_net_device = is_valid_net_device()
	valid_sound_device = is_valid_sound_device()
}

func remove_key(key string) {
	if _, ok := collected_data[key]; ok {
		delete(collected_data, key)
	}
}

func collect_date(key string) {
	t := time.Now()
	collected_data[key] = fmt.Sprintf("%s %s", icons[key], t.Format("Mon Jan _2"))
}

func collect_time(key string) {
	t := time.Now()
	collected_data[key] = fmt.Sprintf("%s %s", icons[key], t.Format("15:04"))
}

func adjust_width(s string, width int) string {
	s = strings.TrimLeft(s, " ")
	if len(s) < width {
		diff := width - len(s)
		for i := 0; i < diff; i++ {
			s += " "
		}
	}
	return s
}

func format_bytes(b int) string {
	kb := float32(b) / 1000.0
	if kb < 1 {
		return adjust_width(fmt.Sprintf("%3dB", b), 4)
	}
	mb := kb / 1000.0
	if mb < 1 {
		return adjust_width(fmt.Sprintf("%3.1fK", kb), 4)
	}
	gb := mb / 1000.0
	if gb < 1 {
		return adjust_width(fmt.Sprintf("%3.1fM", mb), 4)
	}
	tb := gb / 1000.0
	if tb < 1 {
		return adjust_width(fmt.Sprintf("%3.1fG", gb), 4)
	}

	return adjust_width(fmt.Sprintf("%3.1fT", tb), 4)
}

func collect_network(rxkey, txkey string) {
	if !valid_net_device {
		return
	}

	// from: https://github.com/schachmat/gods/blob/master/gods.go
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		remove_key(rxkey)
		remove_key(txkey)
		return
	}
	defer file.Close()

	var void = 0 // target for unused values
	var dev, rx, tx, rxNow, txNow = "", 0, 0, 0, 0
	var scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		_, err = fmt.Sscanf(scanner.Text(), "%s %d %d %d %d %d %d %d %d %d",
			&dev, &rx, &void, &void, &void, &void, &void, &void, &void, &tx)

		if dev[:len(dev)-1] == dwm_config.Network_Interface {
			rxNow += rx
			txNow += tx
		}
	}

	defer func() { rx_old, tx_old = rxNow, txNow }()
	rxdata := fmt.Sprintf("%s %s", icons[rxkey], format_bytes(rxNow-rx_old))
	txdata := fmt.Sprintf("%s %s", icons[txkey], format_bytes(txNow-tx_old))
	collected_data[rxkey] = rxdata
	collected_data[txkey] = txdata
}

func collect_brightness(key string) {
    // For IBM T60
	//actual, err := ioutil.ReadFile("/sys/class/backlight/acpi_video0/actual_brightness")
    // For Thinkpad X220
	actual, err := ioutil.ReadFile("/sys/class/backlight/intel_backlight/actual_brightness")
	if err != nil {
		remove_key(key)
		return
	}

    // For IBM T60
	//max, err := ioutil.ReadFile("/sys/class/backlight/acpi_video0/max_brightness")
    // For Thinkpad X220
	max, err := ioutil.ReadFile("/sys/class/backlight/intel_backlight/max_brightness")
	if err != nil {
		remove_key(key)
		return
	}

	var actual_br, max_br int
	_, err = fmt.Sscanf(string(actual), "%d", &actual_br)
	if err != nil {
		remove_key(key)
		return
	}
	_, err = fmt.Sscanf(string(max), "%d", &max_br)
	if err != nil {
		remove_key(key)
		return
	}

	cur := 100 * actual_br / max_br
	collected_data[key] = fmt.Sprintf("%s%3d%%", icons[key], cur)
}

func collect_temperature(key string) {
	temp1, err := ioutil.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		remove_key(key)
		return
	}

	temp2, err := ioutil.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		remove_key(key)
		return
	}

	var temp1_val, temp2_val int
    var icon string

	_, err = fmt.Sscanf(string(temp1), "%d", &temp1_val)
	if err != nil {
		remove_key(key)
		return
	}
	_, err = fmt.Sscanf(string(temp2), "%d", &temp2_val)
	if err != nil {
		remove_key(key)
		return
	}

	temp := (temp1_val + temp2_val) / 2 / 1000

	if temp > 90 {
		icon = icons["temp100"]
	} else if temp > 50 {
		icon = icons["temp50"]
	} else if temp > 25 {
		icon = icons["temp25"]
	} else if temp > 10 {
		icon = icons["temp0"]
    }

	collected_data[key] = fmt.Sprintf("%s%3dC", icon, temp)
}

func collect_power(key string) {
	out, err := exec.Command("acpi", "-b").Output()
	if err != nil || len(out) == 0 {
		remove_key(key)
		return
	}

	output := string(out)

	split := strings.Split(output, " ")
	charge := split[3][:len(split[3])-1]

	var value int
	var icon string
	_, err = fmt.Sscanf(charge, "%d", &value)
	if err != nil {
		remove_key(key)
		return
	}

	if value > 75 {
		icon = icons["battery100"]
	} else if value > 50 {
		icon = icons["battery75"]
	} else if value > 25 {
		icon = icons["battery50"]
	} else if value > 10 {
		icon = icons["battery25"]
	} else {
		icon = icons["battery0"]
	}

	collected_data[key] = fmt.Sprintf("%s %s", icon, charge)
}

func collect_ram(key string) {
	// from: https://github.com/schachmat/gods/blob/master/gods.go
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		remove_key(key)
		return
	}
	defer file.Close()

	// done must equal the flag combination (0001 | 0010 | 0100 | 1000) = 15
	var total, used, done = 0, 0, 0
	for info := bufio.NewScanner(file); done != 15 && info.Scan(); {
		var prop, val = "", 0
		if _, err = fmt.Sscanf(info.Text(), "%s %d", &prop, &val); err != nil {
			remove_key(key)
			return
		}

		switch prop {
		case "MemTotal:":
			total = val
			used += val
			done |= 1
		case "MemFree:":
			used -= val
			done |= 2
		case "Buffers:":
			used -= val
			done |= 4
		case "Cached:":
			used -= val
			done |= 8
		}
	}

	ram := used * 100 / total
	collected_data[key] = fmt.Sprintf("%s%2d%%", icons[key], ram)
}

func collect_cpu(key string) {
	// from: https://github.com/schachmat/gods/blob/master/gods.go
	var load float32
	loadavg, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		remove_key(key)
		return
	}

	_, err = fmt.Sscanf(string(loadavg), "%f", &load)
	if err != nil {
		remove_key(key)
		return
	}

	cpu := int(load * 100.0 / float32(cores))
	collected_data[key] = fmt.Sprintf("%s%3d%%", icons[key], cpu)
}

func collect_volume(key string) {
	if !valid_sound_device {
		return
	}

	device_id, err := strconv.Atoi(dwm_config.Sound_Device)
	if err != nil {
		remove_key(key)
		return
	}

	out, err := exec.Command("pactl", "list", "sinks").Output()
	if err != nil {
		remove_key(key)
		return
	}

	output := string(out)
	var trimmed string
	volumes := make([]string, device_id+1, device_id+1)
	muted := make([]bool, device_id+1, device_id+1)
	volumes_index := 0
	mute_index := 0
	re_inside_whtsp := regexp.MustCompile(`[\s\p{Zs}]{2,}`)

	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		trimmed = re_inside_whtsp.ReplaceAllString(strings.TrimLeft(scanner.Text(), " \t\r\n"), " ")
		if strings.HasPrefix(trimmed, "Mute: ") {
			// Mute: no
			muted[mute_index] = (strings.Split(trimmed, " ")[1] == "yes")
			mute_index++
		}
		if strings.HasPrefix(trimmed, "Volume: ") {
			// Volume: front-left: 43055 /  66% / -10.95 dB,   front-right: 43055 /  66% / -10.95 dB
			//fmt.Println(splitted)
			volumes[volumes_index] = strings.Split(trimmed, " ")[4]
			volumes_index++
			if volumes_index > device_id {
				break
			}
		}
	}

	volume, err := strconv.Atoi(volumes[device_id][:len(volumes[device_id])-1])
	if err != nil {
		remove_key(key)
		return
	}

	var icon string
	//muted_char := " "
	if muted[device_id] {
		//muted_char = "M"
		icon = icons["volume_mute"]
	} else {
		if volume > 40 {
			icon = icons["volume_loud"]
		} else {
			icon = icons["volume_low"]
		}
	}

	//collected_data[key] = fmt.Sprintf("%s %s %s", icon, volumes[device_id], muted_char)
	collected_data[key] = fmt.Sprintf("%s %s", icon, volumes[device_id])
}

func collect_stats() {
	collect_network("rx", "tx")
	collect_power("battery")
    collect_volume("volume")
    collect_brightness("brightness")
    collect_temperature("temp")
	collect_cpu("cpu")
	collect_ram("ram")
	collect_date("calendar")
	collect_time("clock")
}

func status_bar() string {
	bar := ""
	var key string
	for i := range keys {
		key = keys[i]
		if data, ok := collected_data[key]; ok {
			bar = fmt.Sprintf("%s %s", bar, data)
		}
	}
	return bar
}

func colored_status_bar() string {
	bar := colors[0].end
	var key string
	num_colors := len(colors)
	color_index := 1
	for i := range keys {
		key = keys[i]
		if data, ok := collected_data[key]; ok {
			bar = fmt.Sprintf("%s %s %s %s", bar, colors[color_index%num_colors].start, colors[color_index%num_colors].end, data)
			color_index++
		}
	}
	return bar
}

func update_status_bar() {
	collect_stats()
	exec.Command("xsetroot", "-name", status_bar()).Run()
}

func reload_status_bar() {
    collect_stats()
	exec.Command("xsetroot", "-name", status_bar()).Run()
}

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("Usage: %s <config file>\n", os.Args[0])
		os.Exit(1)
	}
	config_file = os.Args[1]

	collected_data = make(map[string]string)
	load_config()

	signal_chan := make(chan os.Signal, 1)
	go func() {
		for {
			s := <-signal_chan
			switch s {
			case syscall.SIGHUP:
				fmt.Println("reloading config")
				load_config()
				update_status_bar()
			case syscall.SIGUSR1:
				reload_status_bar()
			default:
				fmt.Println(s)
			}
		}
	}()

	signal.Notify(signal_chan, syscall.SIGHUP, syscall.SIGUSR1)

	for {
		update_status_bar()

		// Sleep until beginning of next second.
		var now = time.Now()
		time.Sleep(now.Truncate(time.Second).Add(time.Second).Sub(now))
	}
}
