# go-settlement

## rule

+ 模仿合约，caller替代合约中的msg.sender
+ 不对map进行直接遍历，通常存放一个数组用于遍历
+ 使用globalMap存放所有合约和接口的对应

## 使用

### ErcToken

ERC2.0代币，主要功能：

+ 查询余额，查询代支付余额
+ Transfer 直接转账
+ Approve和TransferFrom配合使用： 1. A 用Approve 允许B使用 amount金额，2. B 用TransferFrom 从A中转账出去；此情形用于合约中转账给合约

### RoleMgr

角色管理，主要功能：

+ 角色有0，user：1，provider：2，keeper：3； 每个地址只能成为1/2/3中一个
+ 每个address都要调用Register，然后获得系统的index，作为系统地址，此时角色为0
+ 可以质押，Pledge
+ Pledge后可以注册成为keeper和provider
+ admin创建group，每个组有个fsMgr合约地址
  
### FsMgr

存储管理，主要功能：

+ 在CreateFs后，需要在RoleMgr中注册成为user
+ AddOrder加入用户每一个订单
+ SubOrder在订单过期后调用
+ provider费用：根据价格，存储量，以及已存储时长在线下共识后，线上获取；
+ keeper组费用：5%订单费用，4%根据provider获取时获取；1%在订单到期后获取；
+ keeper内部分成：根据调用AddOrder，SubOrder的次数，来分成

## 流程

### pre1

admin创建RoleMgr合约

### pre2

+ admin CreateGroup

### 注册keeper

基于pre1

+ Register获取index；
+ Pledge质押
+ RegisterKeeper注册
+ Withdraw

基于pre2

+ 加入某group

### 注册provider

基于pre

+ Register获取index；
+ Pledge质押
+ RegisterKeeper注册
+ Withdraw

基于pre2

+ 加入某group

### 注册user

基于pre

+ Register获取index；

基于pre2

+ 加入某group

## 增发逻辑

+ 根据当前市场需要支付的额度进行增发额度
+ 根据总时空值确定增发系数


## todo

+ 修复

