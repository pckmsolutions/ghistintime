# GhistInTime (`gh`)

**GhistInTime** (`gh`) is a fast, lightweight global terminal command history tracker written in Go. It automatically intercepts executed bash commands and stores them in a local SQLite database, giving you a searchable, persistent command history complete with timestamps and IDs.

---

## Features

* **Automatic History Collection:** Intercepts every command executed in your shell seamlessly using `PROMPT_COMMAND`.


* **Automatic Deduplication:** Moves duplicate command executions to the latest slot while updating the timestamp.


* **Flexible Formatting:** Display history with optional execution timestamps (`--dt` / `-d`) and database record IDs (`--id` / `-i`).


* **Number Limits (`-n`):** Pass numeric limit shorthands (like `gh -20`) to quickly retrieve recent entries.


* **Local SQLite Backend:** Fast, zero-configuration local database storage (`~/.ghist2.db` by default).



---

## Installation

### 1. Build from Source

Clone the repository and compile the binary:

```bash
git clone https://github.com/pckmsolutions/ghistintime.git
cd ghist
go build -o gh main.go

```

Move the compiled binary to a location in your `PATH` (for example, `/usr/local/bin`):

```bash
sudo mv gh /usr/local/bin/

```

### 2. Configure Shell (`.bashrc`)

To automatically record executed commands to the SQLite database, append the following line to your `~/.bashrc` file:

```bash
export PROMPT_COMMAND='gh add "$(fc -ln -0)"'

```

Reload your shell configuration:

```bash
source ~/.bashrc

```

---

## Usage & Commands

### Display Recent History (Default View)

Running `gh` without arguments dumps the history stored in your database:

```bash
gh

```

### Limiting Results

Use the limit flag or shorthand syntax to output only the last $N$ commands:

```bash
# Display the last 10 commands using the -l flag
gh -l 10

# Direct numeric shorthand syntax
gh -20

```

### Formatting Options

Customize the output table using display flags:

* `-d`, `--dt`: Include the local date and time when the command was recorded.


* `-i`, `--id`: Include the unique database primary key ID.


* `--db`: Specify a custom path to an SQLite database file.



```bash
# Show history with local date/time and IDs
gh -d -i

# Combine flags with a row limit
gh -20 -d -i

```

---

## Migrating from Ghist v1 (`ghist.db`)

If you are upgrading from an older version of `ghist` that used `~/.config/ghist.db`, you can migrate your existing history into the new database schema (`~/.ghist2.db`) using SQLite:

```bash
# 1. Ensure the new database file and table schema are initialized
gh

# 2. Open the new SQLite database
sqlite3 ~/.ghist2.db

# 3. Attach your legacy database file and migrate records
sqlite> ATTACH DATABASE '/home/peter/.config/ghist.db' AS old_db;
sqlite> INSERT INTO ghist(id, command, inserted) SELECT id, TRIM(command), inserted FROM old_db.ghist;
sqlite> DETACH DATABASE old_db;
sqlite> .exit

```

---

## Configuration & Storage

* **Default Database:** `~/.ghist2.db`

* **Custom Database File:** You can override the database target at runtime using the `--db` flag:


```bash
gh --db /path/to/custom_history.db

```



---

## License

MIT License
