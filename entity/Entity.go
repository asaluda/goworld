package entity

import (
	"fmt"
	"reflect"

	"time"

	. "github.com/xiaonanln/goworld/common"
	"github.com/xiaonanln/typeconv"

	timer "github.com/xiaonanln/goTimer"
	"github.com/xiaonanln/goworld/components/dispatcher/dispatcher_client"
	"github.com/xiaonanln/goworld/gwlog"
)

type Entity struct {
	ID       EntityID
	TypeName string
	I        IEntity
	IV       reflect.Value

	rpcDescMap RpcDescMap

	space *Space
	aoi   AOI

	timers map[*timer.Timer]struct{}
}

type IEntity interface {
	OnInit()
	OnCreated()
	OnEnterSpace()
	OnDestroy()
}

func (e *Entity) String() string {
	return fmt.Sprintf("%s<%s>", e.TypeName, e.ID)
}

func (e *Entity) Destroy() {
	gwlog.Info("%s.Destroy.", e)
	if e.space != nil {
		e.space.leave(e)
	}
	e.clearTimers()
	e.I.OnDestroy()
	entityManager.del(e.ID)
}

// Space Operations related to entity

// Interests and Uninterest among entities
func (e *Entity) interest(other *Entity) {
	e.aoi.interest(other)
}

func (e *Entity) uninterest(other *Entity) {
	e.aoi.uninterest(other)
}

func (e *Entity) Neighbors() EntitySet {
	return e.aoi.neighbors
}

// Timer & Callback Management
func (e *Entity) AddCallback(d time.Duration, cb timer.CallbackFunc) {
	var t *timer.Timer
	t = timer.AddCallback(d, func() {
		delete(e.timers, t)
		cb()
	})
	e.timers[t] = struct{}{}
}

func (e *Entity) Post(cb timer.CallbackFunc) {
	e.AddCallback(0, cb)
}

func (e *Entity) AddTimer(d time.Duration, cb timer.CallbackFunc) {
	t := timer.AddTimer(d, cb)
	e.timers[t] = struct{}{}
}

func (e *Entity) clearTimers() {
	for t := range e.timers {
		t.Cancel()
	}
	e.timers = map[*timer.Timer]struct{}{}
}

// Call other entities
func (e *Entity) Call(id EntityID, method string, args ...interface{}) {
	callRemote(id, method, args)
}

func (e *Entity) onCall(methodName string, args []interface{}) {
	defer func() {
		err := recover() // recover from any error during RPC call
		if err != nil {
			gwlog.TraceError("%s.%s%v paniced: %s", e, methodName, args, err)
		}
	}()

	rpcDesc := e.rpcDescMap[methodName]
	methodType := rpcDesc.MethodType

	if rpcDesc.NumArgs != len(args) {
		gwlog.Error("Method %s receives %d arguments, but given %d: %v", methodName, rpcDesc.NumArgs, len(args), args)
		return
	}

	in := make([]reflect.Value, len(args)+1)
	in[0] = reflect.ValueOf(e.I)
	for i, arg := range args {
		argType := methodType.In(i + 1)
		in[i+1] = typeconv.Convert(arg, argType)
	}
	rpcDesc.Func.Call(in)
}

// Register for global service
func (e *Entity) DeclareService(serviceName string) {
	dispatcher_client.GetDispatcherClientForSend().SendDeclareService(e.ID, serviceName)
}

// Default Handlers
func (e *Entity) OnInit() {
	gwlog.Warn("%s.OnInit not implemented", e)
}

func (e *Entity) OnCreated() {
	gwlog.Debug("%s.OnCreated", e)
}

func (e *Entity) OnEnterSpace() {
	gwlog.Debug("%s.OnEnterSpace >>> %s", e, e.space)
}

func (e *Entity) OnDestroy() {
}