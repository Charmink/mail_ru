package main

import (
	"container/heap"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

var mt sync.Mutex

type Item struct {
	key      uint32
	value    string
	priority int64
	index    int
	capacity uint64 // обем итема

}

type LRUCache struct {
	Length            int
	Cache             map[uint32]string
	IndexCallLastTime map[uint32]*Item
	Queue             PriorityQueue
	MaxCapacity       uint64 //максимальный обьем кэша
	CurrentCapacity   uint64 // текущий обьем кэша

}

type PriorityQueue []*Item // очередь с приоритетом которая будет хранить итемы, сортироваться по времени последнего обращения

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1 // для безопасности
	*pq = old[0 : n-1]
	return item
}

// update изменяет приоритет и значение Item в очереди.
func (pq *PriorityQueue) update(item *Item, priority int64) {
	item.priority = priority
	heap.Fix(pq, item.index)
}

func NewCache(length int) *LRUCache { // в этой функции происходит инициализация кэша
	lruCache := LRUCache{Length: length, Cache: make(map[uint32]string), Queue: make(PriorityQueue, 0),
		IndexCallLastTime: make(map[uint32]*Item), MaxCapacity: math.MaxUint64, CurrentCapacity: 0}
	return &lruCache
}

func (lrucache *LRUCache) Get(key uint32) (string, error) { // функция реализует запрос к итему из кэша если это возможно
	mt.Lock()
	defer mt.Unlock()
	val, inMap := lrucache.Cache[key]
	if !inMap { // если итем отсутствует, возвращаем ошибку
		return "", errors.New("Key not found!")
	} else { // иначе обновляем время обращения к итему и возвращаем значение
		timeNow := time.Now().UnixNano()
		item := lrucache.IndexCallLastTime[key]
		lrucache.Queue.update(item, timeNow)
		return val, nil
	}
}

func (lrucache *LRUCache) Put(key uint32, value string) error { // функция добавляет элемент в кэш если это возможно
	mt.Lock()
	defer mt.Unlock()
	if _, inMap := lrucache.Cache[key]; !inMap {
		if lrucache.MaxCapacity < uint64(len(value)) { // в случае если строка не влезает даже в пустой кэш, то возвращаем ошибку
			return errors.New("This value cant write in LRU cache, capacity of cache lower then value!")
		}
		if lrucache.Queue.Len() < lrucache.Length && lrucache.MaxCapacity-lrucache.CurrentCapacity >= uint64(len(value)) {
			// если новый итем может быть помещен в кэш без удаления элементов из кэша то добавляем его
			lrucache.Cache[key] = value
			timeNow := time.Now().UnixNano()
			newItem := Item{key: key, value: value, priority: timeNow, capacity: uint64(len(value))}
			heap.Push(&lrucache.Queue, &newItem)
			lrucache.IndexCallLastTime[key] = &newItem
			lrucache.CurrentCapacity += uint64(len(value))
			return nil
		} else {
			// иначе удаляем из кэша самые старые по обрпщению итемы пока новый итем не сможет поместиться в кэш
			for lrucache.Queue.Len() >= lrucache.Length || lrucache.MaxCapacity-lrucache.CurrentCapacity < uint64(len(value)) {
				item := heap.Pop(&lrucache.Queue).(*Item)
				delete(lrucache.Cache, item.key)
				delete(lrucache.IndexCallLastTime, item.key)
				lrucache.CurrentCapacity -= item.capacity
			}
			//записываем новый итем в кэш
			lrucache.Cache[key] = value
			timeNow := time.Now().UnixNano()
			newItem := Item{key: key, value: value, priority: timeNow, capacity: uint64(len(value))}
			heap.Push(&lrucache.Queue, &newItem)
			lrucache.IndexCallLastTime[key] = &newItem
			lrucache.CurrentCapacity += uint64(len(value))
			return nil
		}
	} else {
		return errors.New("This key already exists in the table!")
	}
}

func (lrucache *LRUCache) SetCapacity(capacity uint64) { // здесь устанавливается максимальный обьем в байтах для кэша
	lrucache.MaxCapacity = capacity

}

func timeToLiveControl(lrucache *LRUCache, dur uint64) error {
	if dur >= 1 {
		for true {
			if lrucache.Queue.Len() != 0 {
				mt.Lock()
				item := heap.Pop(&lrucache.Queue)
				//fmt.Println(uint64(time.Now().UnixNano()) - uint64(item.(*Item).priority), dur * 1e9)
				if uint64(time.Now().UnixNano())-uint64(item.(*Item).priority) < dur*1e9 {
					heap.Push(&lrucache.Queue, item)
				} else {
					//fmt.Println(item.(*Item))
					delete(lrucache.Cache, item.(*Item).key)
					delete(lrucache.IndexCallLastTime, item.(*Item).key)
					lrucache.CurrentCapacity -= uint64(len(item.(*Item).value))
				}
				mt.Unlock()

			}

			//time.Sleep(time.Second)

		}
	}
	return nil
}

func main() {
	lrucache := NewCache(3)
	lrucache.SetCapacity(100)
	heap.Init(&lrucache.Queue)
	go timeToLiveControl(lrucache, 10)
	lrucache.Put(1, "abcd")
	lrucache.Put(2, "123")
	lrucache.Put(3, "qwer")

	fmt.Println(lrucache.Cache)

	time.Sleep(time.Second * 2)
	fmt.Println(lrucache.Cache)
	lrucache.Get(2)
	lrucache.Get(1)
	time.Sleep(time.Second * 2)
	fmt.Println(lrucache.Cache)
	lrucache.Put(4, "90")
	time.Sleep(2 * time.Second)
	fmt.Println(lrucache.Cache)

}
