# Same

Lightning fals search for file duplicates in given folder

Copyright 2022 Michael Kondrashin mkondrashin@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[![License](https://img.shields.io/badge/License-Apache%202-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Build:
```code
    git clone --depth 1 https://github.com/mpkondrashin/same.git
    cd same
    go build
```

Usage:
```code
    ./same [options] folder [folder...]
```

Available options:
```
  -hash string
    	hash algorithm. Available values: md5, sha1, sha256 (default "md5")
  -log string
    	log file path
  -report string
    	report file path
  -script string
    	remove duplicates script file path (default "rm.sh")
  -verbose
    	verbose mode
```
