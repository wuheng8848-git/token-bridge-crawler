// Package mail 提供邮件发送功能
package mail

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"

	"token-bridge-crawler/internal/storage"

	"gopkg.in/gomail.v2"
)

// Sender 邮件发送器
type Sender struct {
	dialer   *gomail.Dialer
	from     string
	to       []string
	cc       []string
	template *template.Template
}

// Config 邮件配置
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      bool
	From     string
	To       []string
	Cc       []string
	Subject  string
}

// NewSender 创建邮件发送器
func NewSender(cfg Config) *Sender {
	dialer := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)

	tmpl := template.Must(template.New("subject").Parse(cfg.Subject))

	return &Sender{
		dialer:   dialer,
		from:     cfg.From,
		to:       cfg.To,
		cc:       cfg.Cc,
		template: tmpl,
	}
}

// ReportData 邮件报告数据
type ReportData struct {
	Vendor  string
	Date    string
	Summary string
	Details []storage.VendorPriceDetail
}

// SendReport 发送日报
func (s *Sender) SendReport(data ReportData, attachCSV bool) error {
	var subjectBuf bytes.Buffer
	err := s.template.Execute(&subjectBuf, struct {
		Date    string
		Vendor  string
		Summary string
	}{
		Date:    data.Date,
		Vendor:  strings.ToUpper(data.Vendor),
		Summary: truncate(data.Summary, 30),
	})
	if err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", s.to...)
	if len(s.cc) > 0 {
		m.SetHeader("Cc", s.cc...)
	}
	m.SetHeader("Subject", subjectBuf.String())

	body := s.buildEmailBody(data)
	m.SetBody("text/html", body)

	if attachCSV && len(data.Details) > 0 {
		csvData := s.buildCSV(data.Details)
		m.Attach(fmt.Sprintf("%s-%s-pricing.csv", data.Vendor, data.Date),
			gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(csvData)
				return err
			}))
	}

	return s.dialer.DialAndSend(m)
}

func (s *Sender) buildEmailBody(data ReportData) string {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>
body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
.header { background: #f4f4f4; padding: 20px; border-radius: 5px; }
.summary { margin: 20px 0; padding: 15px; background: #e8f4f8; border-left: 4px solid #2196F3; }
.stats { margin: 20px 0; }
.stats table { width: 100%%; border-collapse: collapse; }
.stats th, .stats td { padding: 10px; border: 1px solid #ddd; text-align: left; }
.stats th { background: #f4f4f4; }
.footer { margin-top: 30px; font-size: 12px; color: #666; }
</style>
</head>
<body>
<div class="header">
<h2>%s 刊例价日报</h2>
<p>日期: %s</p>
</div>
<div class="summary">
<h3>AI 总结</h3>
<p>%s</p>
</div>
<div class="stats">
<h3>统计数据</h3>
<table>
<tr><th>指标</th><th>数值</th></tr>
<tr><td>总模型数</td><td>%d</td></tr>
<tr><td>新增模型</td><td>%d</td></tr>
<tr><td>价格变动</td><td>%d</td></tr>
</table>
</div>
<div class="footer">
<p>此邮件由 Token Bridge Crawler 自动生成</p>
<p>生成时间: %s</p>
</div>
</body>
</html>`,
		strings.ToUpper(data.Vendor), data.Date, data.Summary,
		len(data.Details), countByChangeType(data.Details, "new"),
		countByChangeType(data.Details, "updated"),
		time.Now().Format("2006-01-02 15:04:05"))
	return html
}

func (s *Sender) buildCSV(details []storage.VendorPriceDetail) []byte {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	_ = writer.Write([]string{"Model Code", "Input ($/M)", "Output ($/M)", "Change Type", "Previous Output"})

	for _, d := range details {
		prevOutput := ""
		if d.PrevPrice != nil {
			prevOutput = string(d.PrevPrice)
		}

		_ = writer.Write([]string{
			d.ModelCode,
			fmt.Sprintf("%.6f", d.InputUSDPerMillion),
			fmt.Sprintf("%.6f", d.OutputUSDPerMillion),
			d.ChangeType,
			prevOutput,
		})
	}

	writer.Flush()
	return buf.Bytes()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func countByChangeType(details []storage.VendorPriceDetail, changeType string) int {
	count := 0
	for _, d := range details {
		if d.ChangeType == changeType {
			count++
		}
	}
	return count
}
