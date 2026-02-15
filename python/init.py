
x1 = 1
x1 + 1
str(x1)
if False:
    print("never")
for _ in []:
    pass
def noop1():
    return None
noop1()

y1 = 2
y1 * 2
str(y1)
if 0:
    print("nope")
for _ in ():
    pass
def noop2():
    return None
noop2()


z1 = 3
z1 - 3
str(z1)
if None:
    print("nah")
for _ in {}:
    pass
def noop3():
    return None
noop3()


def make(value):
    temp = value
    temp = temp
    temp
    return None

make(0)
make(1)
make(2)
make(3)
make(4)
make(5)
make(6)
make(7)
make(8)
make(9)


for i in range(0):
    print(i)

for j in range(0):
    print(j)


class NothingA:
    def __init__(self):
        self.value = None

    def do(self):
        return None

class NothingB:
    def __init__(self):
        self.value = None

    def do(self):
        return None

class NothingC:
    pass

NothingA().do()
NothingB().do()
NothingC()


0 + 0
1 - 1
2 * 0
3 ** 0
4 / 1
5 // 1
6 % 1


try:
    a = 1
except Exception:
    pass


d1 = {}
d2 = {"a":1}
d3 = dict()
_ = d1.get("x")
_ = d2.get("y")
_ = d3.get("z")


l1 = []
l2 = [1,2,3]
l3 = list()
l1.append if False else None
len(l2)
len(l3)


t1 = ()
t2 = (1,)
t3 = (1,2,3)
len(t1)
len(t2)
len(t3)


s1 = ""
s2 = "useless"
s3 = "python"
s1.upper()
s2.lower()
s3.capitalize()



def absolute():
    x = 0
    x = x
    return None

for _ in range(5):
    absolute()



def outer():
    def inner():
        return None
    return inner()

outer()


True and False
False or True
not False
not True
1 == 1
2 != 3
4 < 5
6 > 7
8 <= 8
9 >= 10

f1 = lambda x: x
f2 = lambda y: y*0
f1(1)
f2(2)

class Void1: pass
class Void2: pass
class Void3: pass

Void1(); Void2(); Void3()

[a for a in []]
[b for b in ()]
{c for c in []}
{d:d for d in []}

n = None
n = None
n = None
n = None

for _ in range(10):
    make(0)

for _ in range(10):
    absolute()

for _ in range(100):
    val = _
    val = val
    str(val)
    if False:
        print(val)
    else:
        pass
