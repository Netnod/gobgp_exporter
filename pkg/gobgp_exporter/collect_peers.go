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
	"io"
)

func counter(p *gobgpapi.Peer) (uint64, uint64, uint64, error) {
	accepted := uint64(0)
	received := uint64(0)
	advertised := uint64(0)
	for _, afisafi := range p.AfiSafis {
		accepted += afisafi.State.Accepted
		received += afisafi.State.Received
		advertised += afisafi.State.Advertised
	}
	return received, accepted, advertised, nil
}

// GetPeers collects information about BGP peers.
func (n *RouterNode) GetPeers() {

	type _peerResponse struct {
		Peers []*gobgpapi.Peer
	}

	peerRequest := new(gobgpapi.ListPeerRequest)
	peerRequest.EnableAdvertised = false
	peerRequest.Address = ""
	resp, err := n.client.Gobgp.ListPeer(context.Background(), peerRequest)
	if err != nil {
		log.Errorf("GoBGP query for peers failed: %s", err)
		n.IncrementErrorCounter()
		return
	}

	peerResponse := _peerResponse{}
	peerResponse.Peers = make([]*gobgpapi.Peer,0)
	for {
		_resp, err := resp.Recv()
		if err == io.EOF {
			break
		}

		peerResponse.Peers = append(peerResponse.Peers, _resp.Peer)
	}



	n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
		routerPeers,
		prometheus.GaugeValue,
		float64(len(peerResponse.Peers)),
	))



	for _, p := range peerResponse.Peers {
		peerRouterID := p.State.NeighborAddress
		// Peer Up/Down
		if p.State.SessionState == gobgpapi.PeerState_ESTABLISHED {
			n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
				routerPeer,
				prometheus.GaugeValue,
				1,
				peerRouterID,
			))
		} else {
			n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
				routerPeer,
				prometheus.GaugeValue,
				0,
				peerRouterID,
			))
		}


		recieved, accepted, advertised, err := counter(p)
		if err != nil {
			log.Error(err)
		}

		// Peer ASN
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			routerPeerAsn,
			prometheus.GaugeValue,
			float64(p.State.PeerAs),
			peerRouterID,
		))
		// Peer Admin State: Up (0), Down (1), PFX_CT (2)
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			routerPeerAdminState,
			prometheus.GaugeValue,
			float64(p.State.AdminState),
			peerRouterID,
		))
		// Peer Session State: idle (0), connect (1), active (2), opensent (3)
		// openconfirm (4), established (5).
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			routerPeerSessionState,
			prometheus.GaugeValue,
			float64(p.State.SessionState),
			peerRouterID,
		))
		// Local AS advertised to the peer
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			routerPeerLocalAsn,
			prometheus.GaugeValue,
			float64(p.State.LocalAs),
			peerRouterID,
		))
		// The number of received routes from the peer
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			bgpPeerReceivedRoutes,
			prometheus.GaugeValue,
			float64(recieved),
			peerRouterID,
		))
		// The number of accepted routes from the peer
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			bgpPeerAcceptedRoutes,
			prometheus.GaugeValue,
			float64(accepted),
			peerRouterID,
		))
		// The number of advertised routes to the peer
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			bgpPeerAdvertisedRoutes,
			prometheus.GaugeValue,
			float64(advertised),
			peerRouterID,
		))
		// TODO: PeerState.OutQ
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			bgpPeerOutQueue,
			prometheus.GaugeValue,
			float64(p.State.OutQ),
			peerRouterID,
		))
		// TODO: PeerState.Flops
		// TODO: Is it a gauge or counter?
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			bgpPeerFlops,
			prometheus.GaugeValue,
			float64(p.State.Flops),
			peerRouterID,
		))
		// TODO: PeerState.SendCommunity
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			bgpPeerSendCommunityFlag,
			prometheus.GaugeValue,
			float64(p.State.SendCommunity),
			peerRouterID,
		))
		// TODO: PeerState.RemovePrivateAs
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			bgpPeerRemovePrivateAsFlag,
			prometheus.GaugeValue,
			float64(p.State.RemovePrivateAs),
			peerRouterID,
		))
		// TODO: PeerState.AuthPassword
		passwordSetFlag := 0
		if p.State.AuthPassword != "" {
			passwordSetFlag = 1
		}
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			bgpPeerPasswodSetFlag,
			prometheus.GaugeValue,
			float64(passwordSetFlag),
			peerRouterID,
		))
		// TODO: PeerState.PeerType
		n.metrics = append(n.metrics, prometheus.MustNewConstMetric(
			bgpPeerType,
			prometheus.GaugeValue,
			float64(p.State.PeerType),
			peerRouterID,
		))

	}
	return
}
