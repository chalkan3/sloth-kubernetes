---
title: Built-in Functions
description: Complete reference of LISP built-in functions for sloth-kubernetes configuration
sidebar_position: 4
---

# Built-in Functions

sloth-kubernetes configuration files use a LISP dialect with powerful built-in functions for dynamic configuration generation.

## Environment Functions

### env / getenv

Get environment variable value.

```lisp
(env "DIGITALOCEAN_TOKEN")
(env "MY_VAR" "default-value")  ; with fallback
(getenv "AWS_REGION")           ; alias for env
```

### env-or

Get environment variable with required fallback.

```lisp
(env-or "DATABASE_URL" "postgres://localhost:5432/mydb")
```

### env?

Check if environment variable exists.

```lisp
(if (env? "PRODUCTION")
    (size "s-4vcpu-8gb")
    (size "s-1vcpu-1gb"))
```

---

## String Functions

### concat / str

Concatenate strings.

```lisp
(concat "prefix-" (env "CLUSTER_NAME") "-suffix")
(str "node-" "1")  ; alias for concat
```

### format

Format string with placeholders (Printf-style).

```lisp
(format "%s-cluster-%d" "production" 1)
; Result: "production-cluster-1"
```

### upper / lower

Convert string case.

```lisp
(upper "hello")  ; "HELLO"
(lower "WORLD")  ; "world"
```

### trim

Remove leading/trailing whitespace.

```lisp
(trim "  hello  ")  ; "hello"
```

### split

Split string into list.

```lisp
(split "a,b,c" ",")  ; ("a" "b" "c")
```

### join

Join list into string.

```lisp
(join (list "a" "b" "c") ",")  ; "a,b,c"
```

### replace

Replace all occurrences in string.

```lisp
(replace "hello-world" "-" "_")  ; "hello_world"
```

### substring

Extract substring.

```lisp
(substring "hello" 0 3)  ; "hel"
(substring "hello" 2)    ; "llo" (to end)
```

---

## Control Flow

### if

Conditional expression.

```lisp
(if (env? "PRODUCTION")
    "s-4vcpu-8gb"
    "s-1vcpu-1gb")
```

### when

Execute body when condition is true.

```lisp
(when (env? "ENABLE_GPU")
  (gpu-type "nvidia-tesla-v100")
  (gpu-count 2))
```

### unless

Execute body when condition is false.

```lisp
(unless (env? "SKIP_BASTION")
  (bastion (enabled true)))
```

### cond

Multiple condition branches.

```lisp
(cond
  ((= env "production") "s-8vcpu-16gb")
  ((= env "staging")    "s-4vcpu-8gb")
  (true                 "s-2vcpu-4gb"))
```

### default

Return fallback if value is nil or empty.

```lisp
(default (env "REGION") "nyc3")
```

### or / and / not

Boolean operations.

```lisp
(or (env? "AWS_TOKEN") (env? "DO_TOKEN"))
(and (env? "PRODUCTION") (env? "ENABLE_HA"))
(not (env? "DISABLE_VPN"))
```

---

## Comparison

### eq / =

Equality comparison.

```lisp
(eq (env "ENV") "production")
(= (env "COUNT") "3")
```

### Comparison Operators

`!=`, `<`, `>`, `<=`, `>=` - Compare values.

```lisp
(!= (env "ENV") "development")
(< count 10)
(> workers 3)
(<= replicas 5)
(>= memory 8)
```

---

## Arithmetic

### + / - / * / /

Basic arithmetic.

```lisp
(+ 1 2 3)      ; 6
(- 10 3)       ; 7
(* 2 3 4)      ; 24
(/ 10 2)       ; 5
```

### mod

Modulo operation.

```lisp
(mod 10 3)  ; 1
```

---

## Encoding & Hashing

### base64-encode / base64-decode

Base64 encoding/decoding.

```lisp
(base64-encode "secret")
; "c2VjcmV0"

(base64-decode "c2VjcmV0")
; "secret"
```

### sha256

SHA-256 hash.

```lisp
(sha256 "mypassword")
; "89e01536ac207279409d4de1e5253e01f4a1769e696db0d6062ca9b8f56767c8"
```

### md5

MD5-style hash (uses SHA-256 truncated).

```lisp
(md5 "data")
```

---

## UUID & Random

### uuid

Generate UUID v4.

```lisp
(uuid)
; "550e8400-e29b-41d4-a716-446655440000"
```

### random-string

Generate random alphanumeric string.

```lisp
(random-string)      ; 16 chars default
(random-string 32)   ; custom length
```

---

## Time & Date

### now

Current UTC time in RFC3339 format.

```lisp
(now)
; "2024-01-15T10:30:00Z"
```

### timestamp

Current Unix timestamp.

```lisp
(timestamp)
; 1705315800
```

### date

Current date with custom format.

```lisp
(date)                    ; "2024-01-15"
(date "2006/01/02")       ; "2024/01/15"
(date "Jan 02, 2006")     ; "Jan 15, 2024"
```

### time

Current time with custom format.

```lisp
(time)                ; "10:30:00"
(time "15:04")        ; "10:30"
```

---

## System Information

### hostname

Get current hostname.

```lisp
(hostname)
; "my-workstation"
```

### user

Get current username.

```lisp
(user)
; "admin"
```

### home

Get user home directory.

```lisp
(home)
; "/home/admin"
```

### cwd

Get current working directory.

```lisp
(cwd)
; "/projects/my-cluster"
```

---

## File Operations

### read-file

Read file contents.

```lisp
(read-file "~/.ssh/id_ed25519.pub")
(read-file "/etc/hostname")
```

### file-exists?

Check if file exists.

```lisp
(if (file-exists? "~/.kube/config")
    (read-file "~/.kube/config")
    "")
```

### dirname / basename

Path manipulation.

```lisp
(dirname "/path/to/file.txt")   ; "/path/to"
(basename "/path/to/file.txt")  ; "file.txt"
```

### expand-path

Expand path with ~ and make absolute.

```lisp
(expand-path "~/configs")
; "/home/user/configs"
```

---

## Shell Execution

### shell

Execute shell command and return output.

```lisp
(shell "whoami")
(shell "kubectl config current-context")
(shell "curl -s https://api.ipify.org")
```

**Note**: Dangerous commands are blocked for safety.

---

## Variables

### let

Define local variables in a scope.

```lisp
(let ((region "nyc3")
      (size "s-4vcpu-8gb"))
  (node-pool
    (name "workers")
    (region region)
    (size size)))
```

### var

Get variable value.

```lisp
(var "my-variable")
```

### set

Set variable value.

```lisp
(set "cluster-name" "production")
```

---

## List Operations

### list

Create a list.

```lisp
(list "a" "b" "c")
```

### first / rest

Get first element or remaining elements.

```lisp
(first (list 1 2 3))  ; 1
(rest (list 1 2 3))   ; (2 3)
```

### nth

Get element at index.

```lisp
(nth (list "a" "b" "c") 1)  ; "b"
```

### len

Get length of list or string.

```lisp
(len (list 1 2 3))  ; 3
(len "hello")       ; 5
```

### append

Combine lists.

```lisp
(append (list 1 2) (list 3 4))  ; (1 2 3 4)
```

### range

Generate number sequence.

```lisp
(range 5)         ; (0 1 2 3 4)
(range 1 5)       ; (1 2 3 4)
(range 0 10 2)    ; (0 2 4 6 8)
```

---

## Type Checking

### string? / number? / bool? / list? / nil? / empty?

Check value types.

```lisp
(string? "hello")     ; true
(number? 42)          ; true
(bool? true)          ; true
(list? (list 1 2))    ; true
(nil? nil)            ; true
(empty? "")           ; true
(empty? (list))       ; true
```

---

## Type Conversion

### to-string / to-int / to-bool

Convert between types.

```lisp
(to-string 42)        ; "42"
(to-int "42")         ; 42
(to-bool "true")      ; true
(to-bool 1)           ; true
```

---

## Regular Expressions

### match

Extract matches from string.

```lisp
(match "v([0-9]+)\\.([0-9]+)" "v1.28")
; ("v1.28" "1" "28")
```

### match?

Check if pattern matches.

```lisp
(match? "^v[0-9]+" "v1.28")  ; true
(match? "^v[0-9]+" "1.28")   ; false
```

---

## Practical Examples

### Dynamic Cluster Configuration

```lisp
(cluster
  (metadata
    (name (concat "k8s-" (env "ENV" "dev")))
    (environment (env "ENV" "development")))

  (providers
    (digitalocean
      (enabled true)
      (token (env "DIGITALOCEAN_TOKEN"))
      (region (env-or "DO_REGION" "nyc3"))))

  (node-pools
    (pool
      (name "masters")
      (count (if (= (env "ENV") "production") 3 1))
      (size (if (= (env "ENV") "production")
                "s-4vcpu-8gb"
                "s-2vcpu-4gb")))))
```

### Reading SSH Keys from File

```lisp
(providers
  (digitalocean
    (ssh-keys
      (if (file-exists? "~/.ssh/id_ed25519.pub")
          (read-file "~/.ssh/id_ed25519.pub")
          (read-file "~/.ssh/id_rsa.pub")))))
```

### Generating Unique Names

```lisp
(metadata
  (name (concat "cluster-" (substring (uuid) 0 8)))
  (created-at (now)))
```

### Environment-Based Sizing

```lisp
(let ((env (env-or "CLUSTER_ENV" "development")))
  (node-pools
    (pool
      (name "workers")
      (count (cond
               ((= env "production") 10)
               ((= env "staging") 5)
               (true 2)))
      (size (cond
              ((= env "production") "s-8vcpu-16gb")
              ((= env "staging") "s-4vcpu-8gb")
              (true "s-2vcpu-4gb"))))))
```
