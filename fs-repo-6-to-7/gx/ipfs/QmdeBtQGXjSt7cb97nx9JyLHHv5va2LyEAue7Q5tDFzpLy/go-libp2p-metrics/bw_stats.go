package metrics

import (
	flow "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmQFXpvKpF34dK9HcE7k8Ksk8V4BwWYZtdEcjzu5aUgRVr/go-flow-metrics"
	protocol "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	peer "github.com/ipfs/fs-repo-migrations/fs-repo-6-to-7/gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
)

type Stats struct {
	TotalIn  int64
	TotalOut int64
	RateIn   float64
	RateOut  float64
}

type BandwidthCounter struct {
	totalIn  flow.Meter
	totalOut flow.Meter

	protocolIn  flow.MeterRegistry
	protocolOut flow.MeterRegistry

	peerIn  flow.MeterRegistry
	peerOut flow.MeterRegistry
}

func NewBandwidthCounter() *BandwidthCounter {
	return new(BandwidthCounter)
}

func (bwc *BandwidthCounter) LogSentMessage(size int64) {
	bwc.totalOut.Mark(uint64(size))
}

func (bwc *BandwidthCounter) LogRecvMessage(size int64) {
	bwc.totalIn.Mark(uint64(size))
}

func (bwc *BandwidthCounter) LogSentMessageStream(size int64, proto protocol.ID, p peer.ID) {
	bwc.protocolOut.Get(string(proto)).Mark(uint64(size))
	bwc.peerOut.Get(string(p)).Mark(uint64(size))
}

func (bwc *BandwidthCounter) LogRecvMessageStream(size int64, proto protocol.ID, p peer.ID) {
	bwc.protocolIn.Get(string(proto)).Mark(uint64(size))
	bwc.peerIn.Get(string(p)).Mark(uint64(size))
}

func (bwc *BandwidthCounter) GetBandwidthForPeer(p peer.ID) (out Stats) {
	inSnap := bwc.peerIn.Get(string(p)).Snapshot()
	outSnap := bwc.peerOut.Get(string(p)).Snapshot()

	return Stats{
		TotalIn:  int64(inSnap.Total),
		TotalOut: int64(outSnap.Total),
		RateIn:   inSnap.Rate,
		RateOut:  outSnap.Rate,
	}
}

func (bwc *BandwidthCounter) GetBandwidthForProtocol(proto protocol.ID) (out Stats) {
	inSnap := bwc.protocolIn.Get(string(proto)).Snapshot()
	outSnap := bwc.protocolOut.Get(string(proto)).Snapshot()

	return Stats{
		TotalIn:  int64(inSnap.Total),
		TotalOut: int64(outSnap.Total),
		RateIn:   inSnap.Rate,
		RateOut:  outSnap.Rate,
	}
}

func (bwc *BandwidthCounter) GetBandwidthTotals() (out Stats) {
	inSnap := bwc.totalIn.Snapshot()
	outSnap := bwc.totalOut.Snapshot()

	return Stats{
		TotalIn:  int64(inSnap.Total),
		TotalOut: int64(outSnap.Total),
		RateIn:   inSnap.Rate,
		RateOut:  outSnap.Rate,
	}
}
