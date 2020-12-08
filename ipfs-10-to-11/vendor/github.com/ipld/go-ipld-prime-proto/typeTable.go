package dagpb

// Type is a struct embeding a NodePrototype/Type for every Node implementation in this package.
// One of its major uses is to start the construction of a value.
// You can use it like this:
//
// 		dagpb.Type.YourTypeName.NewBuilder().BeginMap() //...
//
// and:
//
// 		dagpb.Type.OtherTypeName.NewBuilder().AssignString("x") // ...
//
var Type typeSlab

type typeSlab struct {
	Bytes         _Bytes__Prototype
	Bytes__Repr   _Bytes__ReprPrototype
	Int           _Int__Prototype
	Int__Repr     _Int__ReprPrototype
	Link          _Link__Prototype
	Link__Repr    _Link__ReprPrototype
	PBLink        _PBLink__Prototype
	PBLink__Repr  _PBLink__ReprPrototype
	PBLinks       _PBLinks__Prototype
	PBLinks__Repr _PBLinks__ReprPrototype
	PBNode        _PBNode__Prototype
	PBNode__Repr  _PBNode__ReprPrototype
	RawNode       _RawNode__Prototype
	RawNode__Repr _RawNode__ReprPrototype
	String        _String__Prototype
	String__Repr  _String__ReprPrototype
}
