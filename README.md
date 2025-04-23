# go-settlement

## rule

+ Imitate the contract, caller replaces msg.sender in the contract

+ Do not traverse the map directly, usually store an array for traversal

+ Use globalMap to store the correspondence of all contracts and interfaces

## Use

### ErcToken

ERC2.0 token, main functions:

+ Check balance, check payment balance
+ Transfer direct transfer
+ Approve and TransferFrom are used together: 1. A uses Approve to allow B to use amount, 2. B uses TransferFrom to transfer from A; this situation is used to transfer from contract to contract

### RoleMgr

Role management, main functions:

+ There are 0 roles, user: 1, provider: 2, keeper: 3; each address can only be one of 1/2/3

+ Each address must call Register, and then obtain the system index as the system address, at this time the role is 0

+ Can pledge, Pledge
+ After Pledge, you can register as a keeper and provider
+ Admin creates a group, each group has an fsMgr contract address

### FsMgr

Storage management, main functions:

+ After CreateFs, you need to register as a user in RoleMgr

+ AddOrder adds each order of the user

+ SubOrder is called after the order expires

+ Provider fee: based on price, storage volume, and storage time, after offline consensus, online acquisition;

+ Keeper group fee: 4% order fee, 3% obtained when the provider is obtained; 1% obtained after the order expires;

+ Keeper internal division: divided according to the number of calls to AddOrder and SubOrder

## Process

### pre1

admin creates RoleMgr contract

### pre2

+ admin CreateGroup

### Register keeper

Based on pre1

+ Register to obtain index;

+ Pledge pledge
+ RegisterKeeper registration
+ Withdraw

Based on pre2

+ Join a group

### Register provider

Based on pre

+ Register to get index;
+ Pledge
+ RegisterKeeper to register
+ Withdraw

Based on pre2

+ Join a group

### Register user

Based on pre

+ Register to get index;

Based on pre2

+ Join a group

## Issuance logic

+ Increase the amount according to the amount that the current market needs to pay
+ Determine the issuance coefficient according to the total time and space value

## todo

+ Fix
