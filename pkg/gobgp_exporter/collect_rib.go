// Copyright 2018 Paul Greenberg (greenpau@outlook.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporter

import (
	gobgpapi "github.com/osrg/gobgp/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"golang.org/x/net/context"
	"strings"
	"io"
)

// GetRibCounters collects BGP routing information base (RIB) related metrics.
func (n *RouterNode) GetRibCounters() {
	if n.connected == false {
		return
	}
	for _, resourceTypeName := range gobgpapi.TableType_name {
		for _, secondaryAddressFamilyName := range gobgpapi.Family_Safi_name {
			if !n.connected {
				continue
			}

			if _, exists := n.resourceTypes[resourceTypeName]; !exists {
				continue
			}
			if _, exists := n.addressFamilies[secondaryAddressFamilyName]; !exists {
				continue
			}

			for _, addressFamilyValue := range gobgpapi.Family_Afi_value {

				var resourceType gobgpapi.TableType
				switch resourceTypeName {
				case "GLOBAL":
					resourceType = gobgpapi.TableType_GLOBAL
				case "LOCAL":
					resourceType = gobgpapi.TableType_LOCAL
				default:
					continue
				}

				ribRequest := new(gobgpapi.ListPathRequest)
				ribRequest.TableType = resourceType
				family := gobgpapi.Family{
					Afi : gobgpapi.Family_Afi(addressFamilyValue),
					Safi : gobgpapi.Family_Safi(gobgpapi.Family_Safi_value[secondaryAddressFamilyName]),
				}
				ribRequest.Family = &family

				pathStream, err := n.client.Gobgp.ListPath(context.Background(), ribRequest)
				if err != nil {
					log.Errorf("GoBGP query failed for resource type %s for %s address family: %s", resourceTypeName, secondaryAddressFamilyName, err)
					n.IncrementErrorCounter()
					continue
				}

				rib := make([]*gobgpapi.Destination, 0)
				for {
					_path, err := pathStream.Recv()
					if err == io.EOF {
						break
					} else if err != nil {
						log.Error(err)
					}
					rib = append(rib, _path.Destination)
				}

				log.Debugf("GoBGP RIB size for %s/%s: %d", resourceTypeName, secondaryAddressFamilyName, len(rib))
				//spew.Dump(len(rib.Destinations))
				n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
					routerRibDestinations,
					prometheus.GaugeValue,
					float64(len(rib)),
					strings.ToLower(resourceTypeName),
					strings.ToLower(secondaryAddressFamilyName),
				))
			}
		}
	}
	return
}
