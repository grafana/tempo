local array = import '../array.libsonnet';
local test = import 'github.com/jsonnet-libs/testonnet/main.libsonnet';

local arr = std.range(0, 10);
local mixedArr = ['a', 1, 'b', 2, 'c', 3, 4, 5, 6, 7, 8, 9, 'd', 'e', 'f', 'g', 'h', 'i', 'j'];

test.new(std.thisFile)

+ test.case.new(
  name='first two',
  test=test.expect.eq(
    actual=array.slice(
      arr,
      index=0,
      end=2,
    ),
    expected=[0, 1],
  )
)
+ test.case.new(
  name='last two',
  test=test.expect.eq(
    actual=array.slice(
      arr,
      index=1,
      end=3,
    ),
    expected=[1, 2],
  )
)
+ test.case.new(
  name='until end',
  test=test.expect.eq(
    actual=array.slice(
      arr,
      index=1
    ),
    expected=[1, 2, 3, 4, 5, 6, 7, 8, 9, 10],
  )
)
+ test.case.new(
  name='from beginning',
  test=test.expect.eq(
    actual=array.slice(
      arr,
      index=0,
      end=2
    ),
    expected=[0, 1],
  )
)
+ test.case.new(
  name='negative start',
  test=test.expect.eq(
    actual=array.slice(
      arr,
      index=-2
    ),
    expected=[9, 10],
  )
)
+ test.case.new(
  name='negative end',
  test=test.expect.eq(
    actual=array.slice(
      arr,
      index=0,
      end=-1
    ),
    expected=[0, 1, 2, 3, 4, 5, 6, 7, 8, 9],
  )
)
+ test.case.new(
  name='step',
  test=test.expect.eq(
    actual=array.slice(
      arr,
      index=0,
      end=5,
      step=2
    ),
    expected=[0, 2, 4],
  )
)

+ test.case.new(
  name='chunkArray',
  test=test.expect.eq(
    actual=array.chunkArray(mixedArr, maxSize=3),
    expected=[['a', 1, 'b'], [2, 'c', 3], [4, 5, 6], [7, 8, 9], ['d', 'e', 'f'], ['g', 'h', 'i'], ['j']],
  )
)
+ test.case.new(
  name='chunkArray - maxSize is 2',
  test=test.expect.eq(
    actual=array.chunkArray(mixedArr, maxSize=2),
    expected=[['a', 1], ['b', 2], ['c', 3], [4, 5], [6, 7], [8, 9], ['d', 'e'], ['f', 'g'], ['h', 'i'], ['j']],
  )
)
+ test.case.new(
  name='chunkArray - maxSize is larger than array length',
  test=test.expect.eq(
    actual=array.chunkArray(mixedArr, maxSize=100),
    expected=[mixedArr],
  )
)
+ test.case.new(
  name='chunkArray - maxSize is 1',
  test=test.expect.eq(
    actual=array.chunkArray(mixedArr, maxSize=1),
    expected=[['a'], [1], ['b'], [2], ['c'], [3], [4], [5], [6], [7], [8], [9], ['d'], ['e'], ['f'], ['g'], ['h'], ['i'], ['j']],
  )
)
