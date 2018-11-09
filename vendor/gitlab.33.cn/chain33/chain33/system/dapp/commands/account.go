package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	. "gitlab.33.cn/chain33/chain33/system/dapp/commands/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func AccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Account management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		DumpKeyCmd(),
		GetAccountListCmd(),
		GetBalanceCmd(),
		ImportKeyCmd(),
		NewAccountCmd(),
		SetLabelCmd(),
	)

	return cmd
}

// dump private key
func DumpKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump_key",
		Short: "Dump private key for account address",
		Run:   dumpKey,
	}
	addDumpKeyFlags(cmd)
	return cmd
}

func addDumpKeyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "address of account")
	cmd.MarkFlagRequired("addr")
}

func dumpKey(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	params := types.ReqString{
		Data: addr,
	}
	var res types.ReplyString
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.DumpPrivkey", params, &res)
	ctx.Run()
}

// get accounts of the wallet
func GetAccountListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get account list",
		Run:   listAccount,
	}
	return cmd
}

func listAccount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res rpctypes.WalletAccounts
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetAccounts", nil, &res)
	ctx.SetResultCb(parseListAccountRes)
	ctx.Run()
}

func parseListAccountRes(arg interface{}) (interface{}, error) {
	res := arg.(*rpctypes.WalletAccounts)
	var result AccountsResult
	for _, r := range res.Wallets {
		balanceResult := strconv.FormatFloat(float64(r.Acc.Balance)/float64(types.Coin), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(r.Acc.Frozen)/float64(types.Coin), 'f', 4, 64)
		accResult := &AccountResult{
			Currency: r.Acc.Currency,
			Addr:     r.Acc.Addr,
			Balance:  balanceResult,
			Frozen:   frozenResult,
		}
		result.Wallets = append(result.Wallets, &WalletResult{Acc: accResult, Label: r.Label})
	}
	return result, nil
}

// get balance of an execer
func GetBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Get balance of a account address",
		Run:   balance,
	}
	addBalanceFlags(cmd)
	return cmd
}

func addBalanceFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account addr")
	cmd.MarkFlagRequired("addr")
	cmd.Flags().StringP("exec", "e", "", getExecuterNameString())
	cmd.Flags().IntP("height", "", -1, "block height")
}

func getExecuterNameString() string {
	str := "executer name (only "
	allowExeName := types.AllowUserExec
	nameLen := len(allowExeName)
	for i := 0; i < nameLen; i++ {
		if i > 0 {
			str += ", "
		}
		str += fmt.Sprintf("\"%s\"", string(allowExeName[i]))
	}
	str += " and user-defined type supported)"
	return str
}

func balance(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	execer, _ := cmd.Flags().GetString("exec")
	height, _ := cmd.Flags().GetInt("height")
	err := address.CheckAddress(addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, types.ErrInvalidAddress)
		return
	}
	if execer == "" {
		req := types.ReqAddr{Addr: addr}
		var res rpctypes.AllExecBalance
		ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetAllExecBalance", req, &res)
		ctx.SetResultCb(parseGetAllBalanceRes)
		ctx.Run()
		return
	}
	if ok := types.IsAllowExecName([]byte(execer), []byte(execer)); !ok {
		fmt.Fprintln(os.Stderr, types.ErrExecNameNotAllow)
		return
	}
	stateHash := ""
	if height >= 0 {
		params := types.ReqBlocks{
			Start:    int64(height),
			End:      int64(height),
			IsDetail: false,
		}
		var res rpctypes.Headers
		ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetHeaders", params, &res)
		_, err := ctx.RunResult()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		h := res.Items[0]
		stateHash = h.StateHash
	}

	var addrs []string
	addrs = append(addrs, addr)
	params := types.ReqBalance{
		Addresses: addrs,
		Execer:    execer,
		StateHash: stateHash,
	}
	var res []*rpctypes.Account
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetBalance", params, &res)
	ctx.SetResultCb(parseGetBalanceRes)
	ctx.Run()
}

func parseGetBalanceRes(arg interface{}) (interface{}, error) {
	res := *arg.(*[]*rpctypes.Account)
	balanceResult := strconv.FormatFloat(float64(res[0].Balance)/float64(types.Coin), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(res[0].Frozen)/float64(types.Coin), 'f', 4, 64)
	result := &AccountResult{
		Addr:     res[0].Addr,
		Currency: res[0].Currency,
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	return result, nil
}

func parseGetAllBalanceRes(arg interface{}) (interface{}, error) {
	res := *arg.(*rpctypes.AllExecBalance)
	accs := res.ExecAccount
	result := AllExecBalance{Addr: res.Addr}
	for _, acc := range accs {
		balanceResult := strconv.FormatFloat(float64(acc.Account.Balance)/float64(types.Coin), 'f', 4, 64)
		frozenResult := strconv.FormatFloat(float64(acc.Account.Frozen)/float64(types.Coin), 'f', 4, 64)
		ar := &AccountResult{
			Currency: acc.Account.Currency,
			Balance:  balanceResult,
			Frozen:   frozenResult,
		}
		result.ExecAccount = append(result.ExecAccount, &ExecAccount{Execer: acc.Execer, Account: ar})
	}
	return result, nil
}

// import private key
func ImportKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import_key",
		Short: "Import private key with label",
		Run:   importKey,
	}
	addImportKeyFlags(cmd)
	return cmd
}

func addImportKeyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "private key")
	cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("label", "l", "", "label for private key")
	cmd.MarkFlagRequired("label")
}

func importKey(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	key, _ := cmd.Flags().GetString("key")
	label, _ := cmd.Flags().GetString("label")
	params := types.ReqWalletImportPrivkey{
		Privkey: key,
		Label:   label,
	}
	var res types.WalletAccount
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.ImportPrivkey", params, &res)
	ctx.SetResultCb(parseImportKeyRes)
	ctx.Run()
}

func parseImportKeyRes(arg interface{}) (interface{}, error) {
	res := arg.(*types.WalletAccount)
	accResult := DecodeAccount(res.GetAcc(), types.Coin)
	result := WalletResult{
		Acc:   accResult,
		Label: res.GetLabel(),
	}
	return result, nil
}

// create an account
func NewAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new account with label",
		Run:   createAccount,
	}
	addCreateAccountFlags(cmd)
	return cmd
}

func addCreateAccountFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("label", "l", "", "account label")
	cmd.MarkFlagRequired("label")
}

func createAccount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	label, _ := cmd.Flags().GetString("label")
	params := types.ReqNewAccount{
		Label: label,
	}
	var res types.WalletAccount
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.NewAccount", params, &res)
	ctx.SetResultCb(parseCreateAccountRes)
	ctx.Run()
}

func parseCreateAccountRes(arg interface{}) (interface{}, error) {
	res := arg.(*types.WalletAccount)
	accResult := DecodeAccount(res.GetAcc(), types.Coin)
	result := WalletResult{
		Acc:   accResult,
		Label: res.GetLabel(),
	}
	return result, nil
}

// set label of an account
func SetLabelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set_label",
		Short: "Set label for account address",
		Run:   setLabel,
	}
	addSetLabelFlags(cmd)
	return cmd
}

func addSetLabelFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().StringP("label", "l", "", "account label")
	cmd.MarkFlagRequired("label")
}

func setLabel(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	label, _ := cmd.Flags().GetString("label")
	params := types.ReqWalletSetLabel{
		Addr:  addr,
		Label: label,
	}
	var res types.WalletAccount
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.SetLabl", params, &res)
	ctx.SetResultCb(parseSetLabelRes)
	ctx.Run()
}

func parseSetLabelRes(arg interface{}) (interface{}, error) {
	res := arg.(*types.WalletAccount)
	accResult := DecodeAccount(res.GetAcc(), types.Coin)
	result := WalletResult{
		Acc:   accResult,
		Label: res.GetLabel(),
	}
	return result, nil
}
