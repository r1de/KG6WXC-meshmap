-- MariaDB dump 10.19  Distrib 10.5.19-MariaDB, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: node_map
-- ------------------------------------------------------
-- Server version	10.5.19-MariaDB-0+deb11u2

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `aredn_info`
--

DROP TABLE IF EXISTS `aredn_info`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `aredn_info` (
  `id` varchar(100) NOT NULL,
  `current_stable_version` varchar(100) DEFAULT NULL,
  `current_nightly_version` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `map_info`
--

DROP TABLE IF EXISTS `map_info`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `map_info` (
  `id` varchar(100) NOT NULL,
  `lastPollingRun` datetime DEFAULT NULL,
  `pollingTimeSec` float DEFAULT NULL,
  `nodeTotal` int(11) DEFAULT NULL,
  `garbageReturned` int(11) DEFAULT NULL,
  `highestHops` int(11) DEFAULT NULL,
  `totalPolled` int(11) DEFAULT NULL,
  `numParallelThreads` int(11) DEFAULT NULL,
  `noLocation` int(11) DEFAULT NULL,
  `mappableLinks` int(11) DEFAULT NULL,
  `mappableNodes` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `node_info`
--

DROP TABLE IF EXISTS `node_info`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `node_info` (
  `node` varchar(70) DEFAULT NULL,
  `wlan_ip` varchar(50) DEFAULT NULL,
  `last_seen` datetime DEFAULT NULL,
  `uptime` varchar(50) DEFAULT NULL,
  `loadavg` varchar(128) DEFAULT NULL,
  `model` varchar(50) DEFAULT NULL,
	`hardware` varchar(50) DEFAULT NULL,
  `firmware_version` varchar(50) DEFAULT NULL,
  `ssid` varchar(50) DEFAULT NULL,
  `channel` varchar(50) DEFAULT NULL,
  `chanbw` varchar(50) DEFAULT NULL,
  `tunnel_installed` varchar(50) DEFAULT NULL,
  `active_tunnel_count` varchar(50) DEFAULT NULL,
  `lat` double DEFAULT NULL,
  `lon` double DEFAULT NULL,
  `wifi_mac_address` varchar(50) DEFAULT NULL,
  `api_version` varchar(50) DEFAULT NULL,
  `board_id` varchar(50) DEFAULT NULL,
  `firmware_mfg` varchar(50) DEFAULT NULL,
  `grid_square` varchar(50) DEFAULT NULL,
  `lan_ip` varchar(50) DEFAULT NULL,
  `services` text DEFAULT NULL,
  `location_fix` int(11) DEFAULT NULL,
  `link_info` text DEFAULT NULL,
  `meshRF` varchar(10) DEFAULT NULL,
  `antGain` int(11) DEFAULT NULL,
  `antBeam` int(11) DEFAULT NULL,
  `antDesc` varchar(100) DEFAULT NULL,
  `antBuiltin` varchar(50) DEFAULT NULL,
  `ethInf` varchar(10) DEFAULT NULL,
  `wlanInf` varchar(10) DEFAULT NULL,
  `hopsAway` int(11) DEFAULT NULL,
  `eth3975` int(11) DEFAULT NULL,
  `description` varchar(1024) DEFAULT NULL,
  `freq` varchar(100) DEFAULT NULL,
  `mesh_gateway` varchar(100) DEFAULT NULL,
  `mesh_supernode` varchar(100) DEFAULT NULL,
  UNIQUE KEY `node_info_UN` (`wlan_ip`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2024-06-10  3:32:55
