<p align="center">
    <img src="https://raw.githubusercontent.com/PKief/vscode-material-icon-theme/ec559a9f6bfd399b82bb44393651661b08aaf7ba/icons/folder-markdown-open.svg" align="center" width="30%">
</p>
<p align="center"><h1 align="center">DFS</h1></p>
<p align="center">
	<em><code>Distributed File Storage</code></em>
</p>
<p align="center">
	<img src="https://img.shields.io/github/license/brinestone/dfs?style=default&logo=opensourceinitiative&logoColor=white&color=0080ff" alt="license">
	<img src="https://img.shields.io/github/last-commit/brinestone/dfs?style=default&logo=git&logoColor=white&color=0080ff" alt="last-commit">
	<img src="https://img.shields.io/github/languages/top/brinestone/dfs?style=default&color=0080ff" alt="repo-top-language">
	<img src="https://img.shields.io/github/languages/count/brinestone/dfs?style=default&color=0080ff" alt="repo-language-count">
</p>
<p align="center"><!-- default option, no dependency badges. -->
</p>
<p align="center">
	<!-- default option, no dependency badges. -->
</p>
<br>

##  Table of Contents

- [ Overview](#-overview)
- [ Features](#-features)
- [ Project Structure](#-project-structure)
  - [ Project Index](#-project-index)
- [ Getting Started](#-getting-started)
  - [ Prerequisites](#-prerequisites)
  - [ Installation](#-installation)
  - [ Usage](#-usage)
  - [ Testing](#-testing)
- [ Project Roadmap](#-project-roadmap)
- [ Contributing](#-contributing)
- [ License](#-license)
- [ Acknowledgments](#-acknowledgments)

---
##  Overview

DFS is a distributed file storage application written in Go. It enables multiple nodes to share and store files across a network, providing redundancy and scalability. The system currently uses TCP as its transport protocol for communication and file data transfer between nodes. DFS is designed to be modular, making it easy to extend or swap out components such as transport layers or storage backends in the future.

---

##  Features

- **End-to-End Encryption:** All file transfers are protected using end-to-end encryption based on the Diffie-Hellman key exchange algorithm, ensuring data privacy and security between nodes.
- **Fast File Sharing:** Efficient protocols enable rapid file distribution and retrieval across the network.
- **Redundant Storage:** Files are replicated across multiple nodes to provide fault tolerance and high availability.
- **Modular Architecture:** Easily extend or replace core components such as transport protocols or storage backends.
- **Scalable Design:** Supports seamless scaling as new nodes join the network.
- **Simple CLI Usage:** Intuitive command-line interface for managing files and nodes.

---

##  Project Structure

```sh
└── dfs/
    ├── .github
    │   └── workflows
    ├── Makefile
    ├── README.md
    ├── api
    │   └── api.go
    ├── cmd
    │   ├── lorem.txt
    │   └── main.go
    ├── encdec
    │   ├── encdec.go
    │   └── encdec_test.go
    ├── fs
    │   └── server.go
    ├── go.mod
    ├── go.sum
    ├── p2p
    │   ├── encoding.go
    │   ├── encoding_test.go
    │   ├── tcp_transport.go
    │   ├── tcp_transport_test.go
    │   └── transport.go
    ├── storage
    │   ├── store.go
    │   └── store_test.go
    └── tmp
        └── build-errors.log
```


###  Project Index
<details open>
	<summary><b><code>DFS/</code></b></summary>
	<details> <!-- __root__ Submodule -->
		<summary><b>__root__</b></summary>
		<blockquote>
			<table>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/go.mod'>go.mod</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/go.sum'>go.sum</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/Makefile'>Makefile</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			</table>
		</blockquote>
	</details>
	<details> <!-- .github Submodule -->
		<summary><b>.github</b></summary>
		<blockquote>
			<details>
				<summary><b>workflows</b></summary>
				<blockquote>
					<table>
					<tr>
						<td><b><a href='https://github.com/brinestone/dfs/blob/master/.github/workflows/ci.yml'>ci.yml</a></b></td>
						<td><code>❯ REPLACE-ME</code></td>
					</tr>
					</table>
				</blockquote>
			</details>
		</blockquote>
	</details>
	<details> <!-- encdec Submodule -->
		<summary><b>encdec</b></summary>
		<blockquote>
			<table>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/encdec/encdec_test.go'>encdec_test.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/encdec/encdec.go'>encdec.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			</table>
		</blockquote>
	</details>
	<details> <!-- storage Submodule -->
		<summary><b>storage</b></summary>
		<blockquote>
			<table>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/storage/store_test.go'>store_test.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/storage/store.go'>store.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			</table>
		</blockquote>
	</details>
	<details> <!-- fs Submodule -->
		<summary><b>fs</b></summary>
		<blockquote>
			<table>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/fs/server.go'>server.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			</table>
		</blockquote>
	</details>
	<details> <!-- p2p Submodule -->
		<summary><b>p2p</b></summary>
		<blockquote>
			<table>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/p2p/transport.go'>transport.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/p2p/encoding_test.go'>encoding_test.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/p2p/tcp_transport.go'>tcp_transport.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/p2p/encoding.go'>encoding.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/p2p/tcp_transport_test.go'>tcp_transport_test.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			</table>
		</blockquote>
	</details>
	<details> <!-- api Submodule -->
		<summary><b>api</b></summary>
		<blockquote>
			<table>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/api/api.go'>api.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			</table>
		</blockquote>
	</details>
	<details> <!-- cmd Submodule -->
		<summary><b>cmd</b></summary>
		<blockquote>
			<table>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/cmd/lorem.txt'>lorem.txt</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			<tr>
				<td><b><a href='https://github.com/brinestone/dfs/blob/master/cmd/main.go'>main.go</a></b></td>
				<td><code>❯ REPLACE-ME</code></td>
			</tr>
			</table>
		</blockquote>
	</details>
</details>

---
##  Getting Started

###  Prerequisites

Before getting started with dfs, ensure your runtime environment meets the following requirements:

- **Programming Language:** Go
- **Package Manager:** Go modules


###  Installation

Install dfs using one of the following methods:

**Build from source:**

1. Clone the dfs repository:
```sh
❯ git clone https://github.com/brinestone/dfs
```

2. Navigate to the project directory:
```sh
❯ cd dfs
```

3. Install the project dependencies:


**Using `go modules`** &nbsp; [<img align="center" src="https://img.shields.io/badge/Go-00ADD8.svg?style={badge_style}&logo=go&logoColor=white" />](https://golang.org/)

```sh
❯ go build
```




###  Usage
Run dfs using the following command:
**Using `go modules`** &nbsp; [<img align="center" src="https://img.shields.io/badge/Go-00ADD8.svg?style={badge_style}&logo=go&logoColor=white" />](https://golang.org/)

```sh
❯ go run {entrypoint}
```


###  Testing
Run the test suite using the following command:
**Using `go modules`** &nbsp; [<img align="center" src="https://img.shields.io/badge/Go-00ADD8.svg?style={badge_style}&logo=go&logoColor=white" />](https://golang.org/)

```sh
❯ go test ./...
```


---
##  Project Roadmap

- [X] **`Task 1`**: <strike>Implement feature one.</strike>
- [ ] **`Task 2`**: Implement feature two.
- [ ] **`Task 3`**: Implement feature three.

---

##  Contributing

- **💬 [Join the Discussions](https://github.com/brinestone/dfs/discussions)**: Share your insights, provide feedback, or ask questions.
- **🐛 [Report Issues](https://github.com/brinestone/dfs/issues)**: Submit bugs found or log feature requests for the `dfs` project.
- **💡 [Submit Pull Requests](https://github.com/brinestone/dfs/blob/main/CONTRIBUTING.md)**: Review open PRs, and submit your own PRs.

<details closed>
<summary>Contributing Guidelines</summary>

1. **Fork the Repository**: Start by forking the project repository to your github account.
2. **Clone Locally**: Clone the forked repository to your local machine using a git client.
   ```sh
   git clone https://github.com/brinestone/dfs
   ```
3. **Create a New Branch**: Always work on a new branch, giving it a descriptive name.
   ```sh
   git checkout -b new-feature-x
   ```
4. **Make Your Changes**: Develop and test your changes locally.
5. **Commit Your Changes**: Commit with a clear message describing your updates.
   ```sh
   git commit -m 'Implemented new feature x.'
   ```
6. **Push to github**: Push the changes to your forked repository.
   ```sh
   git push origin new-feature-x
   ```
7. **Submit a Pull Request**: Create a PR against the original project repository. Clearly describe the changes and their motivations.
8. **Review**: Once your PR is reviewed and approved, it will be merged into the main branch. Congratulations on your contribution!
</details>

<details closed>
<summary>Contributor Graph</summary>
<br>
<p align="left">
   <a href="https://github.com{/brinestone/dfs/}graphs/contributors">
      <img src="https://contrib.rocks/image?repo=brinestone/dfs">
   </a>
</p>
</details>

---

##  License

This project is protected under the [SELECT-A-LICENSE](https://choosealicense.com/licenses) License. For more details, refer to the [LICENSE](https://choosealicense.com/licenses/) file.

---

##  Acknowledgments

- List any resources, contributors, inspiration, etc. here.

---
