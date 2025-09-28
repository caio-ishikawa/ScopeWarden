<div align="center">
    <img src="assets/scopewarden.png" width=250 height=250>
</div>

## 💻 Introduction
ScopeWarden is a self-hostable and configurable automated recon tool with a interactive CLI table for going through the results. It allows for flexible automation of recon workflows without relying on any specific tool.

## ✨ Features
- **Run any recon tool:** The yaml configuration file allows you to set any command for the scan to run, and a way to filter results such that only the relevant output gets considered.
- **Run port scans on found assets:** Each found domain from the recon tool can be port scanned, and the configuration allows you to set specific ports to avoid collecting noise. Alternatively, it can run a complete port scan for each found domain.
- **Conditional brute force**: Each tool can be configured to use a brute force tool (e.g. ffuf, gobuster, etc.), which can itself be configured to run based on the technologies found on the domain (e.g. php, wordpress, apache, etc.)
- **Update messages:** Can be configured to send Telegram messages if a new or previously unavailable domain/port becomes available.

## 📦 Setup & Installation
#### Pre-installation Setup
ScopeWarden expects some environment variables to be set before installing:
- **SCOPEWARDEN_CONFIG:** Should be an absolute path to the configuration yaml file.
- **SCOPEWARDEN_TELEGRAM_API_KEY:** Telegram bot API key. Only necessary if notification is set to true in the configuration file.
- **SCOPEWARDEN_TELEGRAM_CHAT_ID:** Telegram chat ID. Only necessary if notification is set to true in the configuration file.

#### Telegram Notifications Setup
In order to reduce dependencies, ScopeWarden relies on your own Telegram bot and chat ID. To set this up, check the following documentation:
    - **Set up bot token:** https://core.telegram.org/bots/features#botfather
    - **To get your chat ID:** https://gist.github.com/nafiesl/4ad622f344cd1dc3bb1ecbe468ff9f8a#get-chat-id-for-a-private-chat

#### Installing Daeomn/API
1. Clone the repository.
2. In the project directory, run `sudo make daemon`. This builds the binary into `/usr/bin`.
3. If you want the daemon to run as a service in Linux, run `sudo make attach-daemon`. This moves the `scopewarden-daemon.service` file to `/etc/systemd/system/` and starts the daemon as a service. The daemon can be set to start on boot by running `sudo systemctl enable scopewarden-daemon.service`. 
4. If the daemon and API were started as a systemd service, check the logs to make sure it is running with: `sudo journalctl -u scopewarden-daemon`.

#### Installing CLI
1. Clone the repository
2. In the project directory, run `sudo make cli`. This builds the binary into `/usr/bin`.
3. Check installation with `scopewarden -h`.

## 🖥️ Dependencies
- [Golang](https://go.dev/)
- [SQLite](https://sqlite.org/): To store scanning results.
- [Nmap](https://nmap.org/): Not necessary if ScopeWarden is not configured to do automated port scans.
- [xclip](https://github.com/astrand/xclip): To copy URLs to the clipboard on Linux distros using X11.
- A cool terminal theme 😎

## ❓ How it works
ScopeWarden works with targets, scopes and domains:
- **Targets:** Consist of a unique name.
- **Scope:** Represents all the scannable URLs for a specific target. A scope can only be related to a single target.
- **Domains:** Represent all the domains found when scanning a particular scope. The subsequent port scans and brute forcing are done to each domain, as configured in the yaml file.

A scan will start by going over the targets and its scopes. For each scope found, it will run the scan based on the configured toolset (see [Configuration](#Configuration)), and update the DB with the found domains and the associated brute forced paths and found ports. 

**NOTE:** In order to avoid a lot of noise in the DB, ScopeWarden will only store and process the root URL. E.g. If the configured tool finds `https://example.com/some/path/to/something`, ScopeWarden will only process `https://example.com`. The rationale behind this is that tools often return multiple paths to the same root URL, and most times these paths are not relevant, and will end up making the end results harder to parse through. This also speeds up the scanning by ignoring duplicate root URLs before processing them.

## 🔧 Configuration
By default, ScopeWarden will not run any tools in the scan. It will continuously loop trying to find the desired configuration yaml file.
The yaml file can contain the folliwng:
#### Global
- **schedule**: Interval in hours for running scans (e.g., `12` runs every 12 hours). If the previous scan took longer than the set schedule , it will run again after it is completed.
- **notify**: `true` or `false`. Enables Telegram notifications.
- **Intensity:** `aggressive` or `conservative`. Aggressive will use a maximum of 30 concurrent processes to parse domains and 15 concurrent processes to conduct the brute force. Conservative will use 10 and 5 respectively. This field defaults to `conservative`.

#### Tools
Multiple tools are allowed to be configured under the `tools` section, each with the following configurations:
- **id:** Unique name for the tool.  
- **command:** CLI command to run. It supports the placeholder `<target>` that gets replaced with the scope in the current scan.  
- **verbose:** `true` or `false`. Enables stderr logging for the tool. Defaults to false if not set.

- **Output Parser:** Configuration to define how the tool's output gets processed. **Note:** The configured regex will match against all the outputs of a tool. The found match will be tested by means of a GET request, and fingerprinted based on the response. If a specific tool outputs more than the found URL in the same line, it is recommended to pipe the output of the tool to `awk` or similar, such that the tool only the desired outputs that can be matched with the regex.
    - **type**: Currently only supports `realtime` option (parse output as it is produced).  
    - **regex**: Regular expression to extract relevant information from the tool output. 

- **Port Scan:** Configuration to define the automated port scan parameters. 
    - **run:** `true` or `false`. Enables port scan for each found domain. Defaults to `false` if not set.
    - **ports:** List of ports to scan (e.g., `21, 22, 53`). If empty or not set, ScopeWarden will run a port scan with no specified ports.   

- **Brute Force:** Brute force attempts are conducted to found domains in the scan, and can be configured to do so conditionally depending on the technologies fingerprinted on the domain. **Note:** Even though brute forces happen concurrently, it is **heavily** encouraged to use smaller/more focused wordlists to keep the scan from taking too long.
    - **run:** `true` or `false`. Enables brute force scans. Defaults to false if not set. 
    - **command:** The fuzzing command. It supports placeholders `<target>` and `<wordlist>` that get replaced with the domain URL in the current scan and the worlist configured in the **conditions** field.  
    - **regex:** Regex to filter valid results from fuzzing output.
    - **conditions:** Optional list of technology-specific wordlists. If empty or not set, this will run the brute force command for **every found domain**:
      - **technology:** Non-case-sensitive target technology to run the scan (e.g., `php`, `wordpress`). If none is set, the brute force scan will be conducted to **every found domain in the scan (not recommended)**.
      - **wordlist:** Path to the wordlist to use for that technology. Expects absolute path.
- **Overrides**: Configures the per-scope overrides for the specific tool.
    - **scope:** URL for the scope (should be the same as the one added via the CLI).
    - **type:** Configured what to override. Can be `port_scan` to override the ports scanned for the given scope,`brute_force` to override the brute force command for the specific scope (e.g. to change the rate-limit on the tool) or`tool` to override the command for the tool itself for the specific scope (e.g. some commands will have flags to allow/disallow subdomains).
    - **command:** Full command to override. Will be parsed the same way as the command it overrides (e.g. Supports `<target>` for the tool override and `<target>` & `<wordlist>` for the brute force override. This will be ignored if overriding the port scan, and it will use the wordlist per-technology as specified in the `brute_force` parameter.
    - **ports:** List of ports to scan for the specific target.

#### Example Configuration
```
global:
  schedule: 12
  nofity: true
  intensity: 'aggressive'

tools:
  - id: gau
    command: 'gau <target>'
    allow_subs: '--subs'
    disallow_subs: ''
    verbose: false 
    port_scan:
      run: true
      ports:
        - 21
        - 22
        - 53
        - 5432
        - 3306
        - 9092
    brute_force:
      run: true
      command: 'ffuf -u <target>/FUZZ -w <wordlist> -s -mc 200 -rate 30'
      regex: '^\/?(?:[\w-]+(?:\.[\w-]+)*\/)*[\w-]+(?:\.[\w-]+)*\/?$'
      conditions:
        - technology: 'php'
          wordlist: '/abolute/path/to/wordlist'
        - technology: 'apache'
          wordlist: '/abolute/path/to/wordlist'
        - technology: 'nginx'
          wordlist: '/abolute/path/to/wordlist'
    parser:
      type: 'realtime'
      regex: '^(https?:\/\/)?([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})(:\d+)?(\/[^\r\n]*)?$'
    overrides:
      - scope: 'nasa.gov'
        type: 'brute_force'
        command: 'ffuf -u <target>/FUZZ -w <wordlist> -s -mc 200 -rate 1'
      - scope: 'nasa.gov'
        type: 'tool'
        command: 'gau <target> --subs'
      - scope: 'nasa.gov'
        type: 'port_scan'
        ports:
          - 22
    
  - id: waymore
    command: 'waymore -i <target>'
    allow_subs: ''
    disallow_subs: '--no-subs'
    verbose: false
    port_scan:
      run: true
      ports:
        - 21
        - 22
        - 53
        - 5432
        - 3306
        - 9092
    parser:
      type: 'realtime'
      regex: '^(https?:\/\/)?([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})(:\d+)?(\/[^\r\n]*)?$'
```

## 🎯 CLI Usage
The CLI allows you to add targets and scopes, as well as view the recon results per target in a interactive table.
```
  -dT string
        Disable target (<target_name>)
  -iS string
        Comma-separated values for scope. First value should be target name, and the following values will be interpreted as scope URLs (<target_name>,<true/false>,<scope_url>)
  -iT string
        Insert target (<target_name>)
  -s    Show stats
  -t string
        Show target stats based on target name (<target_name>)
```

#### Examples 
- Add target:
    ```
    scopewarden -iT NASA
    ```
- Disable target:
    ```
    scopewarden -dT NASA
    ```
- Add scope for target:
    ```
    scopewarden -iS NASA,nasa.gov,something.com,somethingelse.com
    ```
- View table for target:
    ```
    scopewarden -t NASA
    ```
- View scanning daemon stats:
    ```
    scopewarden -s
    ```

#### Navigating interactive table:
The first table displayed when running -t is the domains table. It shows all domains found when running the configured tools and the status code it received when testing the domain. To navigate the table:
- **[J,K]:** Naviating up and down the tables.
- **[H,L]:** Go back and forward 1 page.
- **[P]:** Switch to the ports table. It displays the ports found for the selected domain when running the configured port scan and their respective port states. 
- **[A]:** Switch to the Assets table. It displays the found assets during the configured brute force attepmts.
- **[B]:** Go back to the main table.
- **[C]:** Copy selected domain URL to clipboard.
- **[S+A, S+P]:** Sort by highest numer of assets and ports respectively.
- **[/]:** Open search input. Can filter results by URL.
- **[Enter]:** Open the selected URL in the default browser. Can only be used in the domains or assets table.

All navigation keys are displayed in the helper text below the table.

## Contributing
Anyone is welcomed to point out issues or open PRs for ScopeWarden. Please remember to update the README in the PR when a change requires it.

I would especially welcome changes towards these features:
- **Per-scope rate-limit:** Add a way to configure ScopeWarden to rate-limit requests and brute force attepmts per-scope. Alternatively, add a override option to the tool that will apply a different command based on the scope name.
- **Allow file output parser for tool:** Add output parser type called 'file' which parses tool output file instead of the real time output in stdout. Ideally it would set the output path to `/tmp` and delete it after processing.
- **Search and select target on interactive CLI instead of by flags:** E.g `scopewarden` command renders a table with all targets and lets you select the target for the main table.
- **Web interface**: Add web interface as an alternative to the CLI. I'm not personally interested in this, but I think it would suit other people's workflows a little nicer.
- **Wayland/Hyprland copy-to-clipboard:** Add a way to copy domains to clipboard, since xclip does not work on wayland. As far as I know there isn't a clipboard tool compatible with both X11 and Wayland, so the CLI should be able to tell what the user is running.

## TODO
- [x] Search by domain
- [ ] Show total pages in domains table
- [ ] Test Makefile installation
