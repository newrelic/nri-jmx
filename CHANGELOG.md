[[#]] Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## 2.4.1 (2019-12-02)
### Changed
- Updated the nrjmx path in the definition file

## 2.4.0 (2019-11-18)
### Changed
- Renamed the integration executable from nr-jmx to nri-jmx in order to be consistent with the package naming. **Important Note:** if you have any security module rules (eg. SELinux), alerts or automation that depends on the name of this binary, these will have to be updated.

## 2.3.5 - 2019-11-19
### Added
- Error when a metric is set for the second time

## 2.3.4 - 2019-11-19
### Added
- Error when a metric is set for the second time

## 2.3.3 - 2019-11-18
### Added
- Add nrjmx version dependency to 1.5.2, so jmxterm can be bundled within
  package.

## 2.3.2 - 2019-11-14
### Added
- Remove windows definition from linux package

## 2.3.1 - 2019-11-14
### Added
- Nrjmx tool within the Windows installer
 
## 2.3.0 - 2019-11-13
### Changed
- Updated `nrjmx` version to the more stable latest one.

## 2.2.5 - 2019-10-16
### Fixed
- Windows installer GUIDs

## 2.2.4 - 2019-09-06
### Fixed
- Panic when JMX attributes have commas in them

## 2.2.3 - 2019-09-06
### Fixed
- Broken build

## 2.2.2 - 2019-08-28
### Added
- Custom uri path

## 2.2.1 - 2019-08-28
### Added
- Local entity support

## 2.2.0 - 2019-08-22
### Added
- Windows build support

## 2.1.0 - 2019-06-19
### Added
- Remote JBoss Standalone support
- Updated the SDK

## 2.0.0 - 2019-04-29
### Changed
- Added identity attributes for better uniqueness
- Updated the SDK

## 1.0.4 - 2019-03-19
### Changed
- Include jvm-metrics.yml.sample in package

## 1.0.3 - 2019-03-18
### Added
- Added remote option to use remote URL connections. Format: service:jmx:remoting-jmx://host:port

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
