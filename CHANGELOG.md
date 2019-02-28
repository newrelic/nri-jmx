# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## 1.0.2 - 2019-02-13
### Added
- Added SSL option to Jmx.Open

## 1.0.1 - 2019-02-04
### Fixed
- Updated protocol version

## 1.0.0 - 2018-11-16
### Changed
- Updated to version 1.0.0

## 0.1.8 - 2018-11-09
### Fixed
- Fix error with incorrect metric type interface

## 0.1.7 - 2018-11-01
### Added
- Sample file for Hikaridb

## 0.1.6 - 2018-09-26
### Changed
- Updated sample configuration file with JMX-specific fields

## 0.1.5 - 2018-09-20
### Fixed
- Fixed bug with parsing JMX queries

## 0.1.4 - 2018-09-19
### Added
- Added tomcat-metrics.yml.sample back as an additional sample

## 0.1.3 - 2018-09-18
### Changed
- Removed extra yml files and renamed existing ones with .sample extension

## 0.1.2 - 2018-09-14
### Added
- Logic to enforce a soft limit on the number of metrics that can be collect. If the number of metrics per Entity exceeds this limit the Entity will not be reported to NR.

## 0.1.1 - 2018-09-13
### Changed
- Renamed nr-jmx-config.yml.template to jmx-config.yml.sample
- Renamed nr-jmx-definition.yml to jmx-definition.yml

## 0.1.0 - 2018-07-24
### Added
- Initial version: Includes Metrics and Inventory data
