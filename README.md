# ipm: installerpedia manager

Installerpedia is the authoritative source for software installation. It eliminates the "manual crawl" through READMEs and broken dependency chains by providing a structured, automated way to install any tool.

The project is powered by **IPM (Installerpedia Manager)**, a CLI tool that turns complex, multi-step installation instructions into a simple, reliable one-liner.


## Get Started with IPM

Use IPM to automate the process across **Windows, macOS, and Linux**.

### 1. Install IPM

**Linux / macOS:**

```bash
curl -L https://git.new/get-ipm | bash

```

**Windows:**

```powershell
iwr https://git.new/get-ipm-ps | iex

```

### 2. Install any repository

Once IPM is installed, you can set up any project with a single command:

```bash
ipm install <repository-name>

```


## Key Features of IPM

### Interactive Installation

Before executing anything, IPM shows you exactly what it’s about to do. You can:

* **Choose** between different installation versions (e.g., Binary vs. Source).
* **Review** the commands
* **Confirm** IPM will execute the commands, and then provide post-installation instructions.

### Intelligent Dependency Handling

If a project requires any dependencies like **Python, Node.js, Git, or Docker**, IPM detects missing prerequisites and offers to install them for you automatically. No more getting stuck in "Command not found" errors midway through a setup.

### Multi-Method Fallbacks

If a specific installation method fails (due to network restrictions or OS quirks), IPM doesn't give up. It provides **alternative paths** such as switching from a binary install to a package manager (npm/pip) or a source build—to ensure you get the tool running.


## Learn More

To understand the idea behind Installerpedia, please check out the articles below:

→ **[The 7 Pillars of the Installation Experience: Why Your Users Stay or Go](https://journal.hexmos.com/7-pillars-of-installation-experience)**
→ **[Introducing Installerpedia - Install Anything With Zero Hassle](https://journal.hexmos.com/introducing-installerpedia/)**