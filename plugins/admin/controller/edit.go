package controller

import (
	"github.com/chenhg5/go-admin/context"
	"github.com/chenhg5/go-admin/modules/auth"
	"github.com/chenhg5/go-admin/modules/menu"
	"github.com/chenhg5/go-admin/plugins/admin/modules/constant"
	"github.com/chenhg5/go-admin/plugins/admin/modules/file"
	"github.com/chenhg5/go-admin/plugins/admin/modules/parameter"
	"github.com/chenhg5/go-admin/plugins/admin/modules/response"
	"github.com/chenhg5/go-admin/plugins/admin/modules/table"
	"github.com/chenhg5/go-admin/template"
	"github.com/chenhg5/go-admin/template/types"
	"net/http"
	"regexp"
	"strings"
)

func ShowForm(ctx *context.Context) {

	prefix := ctx.Query("prefix")
	panel := table.List[prefix]
	if !panel.GetEditable() {
		// TODO: add css style
		response.PageNotFound(ctx)
		return
	}

	formData, title, description := panel.GetDataFromDatabaseWithId(ctx.Query("id"))

	params := parameter.GetParam(ctx.Request.URL.Query())

	user := auth.Auth(ctx)

	tmpl, tmplName := template.Get(config.THEME).GetTemplate(ctx.Headers(constant.PjaxHeader) == "true")
	buf := template.Excecute(tmpl, tmplName, user, types.Panel{
		Content: template.Get(config.THEME).Form().
			SetContent(formData).
			SetPrefix(config.PREFIX).
			SetUrl(config.Url("/edit/" + prefix)).
			SetToken(auth.TokenHelper.AddToken()).
			SetInfoUrl(config.Url("/info/" + prefix + regexp.MustCompile(`&id=[0-9]+`).ReplaceAllString(params.GetRouteParamStr(), ""))).
			SetHeader(panel.GetForm().HeaderHtml).
			SetFooter(panel.GetForm().FooterHtml).
			GetContent(),
		Description: description,
		Title:       title,
	}, config, menu.GetGlobalMenu(user).SetActiveClass(strings.Replace(ctx.Path(), config.PREFIX, "", 1)))
	ctx.Html(http.StatusOK, buf.String())
}

func EditForm(ctx *context.Context) {
	prefix := ctx.Query("prefix")
	panel := table.List[prefix]
	if !panel.GetEditable() {
		response.PageNotFound(ctx)
		return
	}
	token := ctx.FormValue("_t")

	if !auth.TokenHelper.CheckToken(token) {
		response.BadRequest(ctx, "edit fail")
		return
	}

	form := ctx.Request.MultipartForm

	menu.GlobalMenu.SetActiveClass(strings.Replace(ctx.Path(), config.PREFIX, "", 1))

	// process uploading files, only support local storage
	if len(form.File) > 0 {
		_, _ = file.GetFileEngine("local").Upload(form)
	}

	var err error
	if prefix == "manager" { // manager edit
		if err = editManager(form.Value); err != nil {
			response.Error(ctx, err.Error())
			return
		}
	} else if prefix == "roles" { // role edit
		if err = editRole(form.Value); err != nil {
			response.Error(ctx, err.Error())
			return
		}
	} else {
		val := form.Value
		for _, f := range panel.GetForm().FormList {
			if f.Editable {
				continue
			}
			if len(val[f.Field]) > 0 && f.Field != "id" {
				response.BadRequest(ctx, "field["+f.Field+"]is not editable")
				return
			}
		}
		panel.UpdateDataFromDatabase(form.Value)
	}

	table.RefreshTableList()

	previous := ctx.FormValue("_previous_")
	prevUrlArr := strings.Split(previous, "?")
	params := parameter.GetParamFromUrl(previous)

	previous = config.Url("/info/" + prefix + regexp.MustCompile(`&id=[0-9]+`).ReplaceAllString(params.GetRouteParamStr(), ""))
	editUrl := config.Url("/info/" + prefix + "/edit" + params.GetRouteParamStr())
	newUrl := config.Url("/info/" + prefix + "/new" + params.GetRouteParamStr())
	deleteUrl := config.Url("/delete/" + prefix)

	panelInfo := panel.GetDataFromDatabase(prevUrlArr[0], params)

	dataTable := template.Get(config.THEME).
		DataTable().
		SetInfoList(panelInfo.InfoList).
		SetThead(panelInfo.Thead).
		SetNewUrl(newUrl)

	if panelInfo.Editable {
		dataTable.SetEditUrl(editUrl)
	}
	if panelInfo.Deletable {
		dataTable.SetDeleteUrl(deleteUrl)
	}

	box := template.Get(config.THEME).Box().
		SetBody(dataTable.GetContent()).
		SetHeader(dataTable.GetDataTableHeader() + panel.GetInfo().HeaderHtml).
		WithHeadBorder(false).
		SetFooter(panel.GetInfo().FooterHtml + panelInfo.Paginator.GetContent()).
		GetContent()

	user := auth.Auth(ctx)

	tmpl, tmplName := template.Get(config.THEME).GetTemplate(true)
	buf := template.Excecute(tmpl, tmplName, user, types.Panel{
		Content:     box,
		Description: panelInfo.Description,
		Title:       panelInfo.Title,
	}, config, menu.GetGlobalMenu(user).SetActiveClass(strings.Replace(previous, config.PREFIX, "", 1)))

	ctx.Html(http.StatusOK, buf.String())
	ctx.AddHeader(constant.PjaxUrlHeader, previous)
}
