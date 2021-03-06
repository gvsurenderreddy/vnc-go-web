package main // github.com/gitinsky/vnc-go-web

import (
	"net/http"
    "encoding/base64"
    "net/url"
    "strings"
    "crypto/tls"
    "io"
    "io/ioutil"
    "fmt"
    "net"
)

func (p *Responder) serveLogin() {
	p.r.ParseForm()

    serverNum := strings.Trim(p.r.PostFormValue("servernum"), " \r\n")
    if serverNum == "" {
        http.Redirect(p.w, p.r, *cfg.baseuri+"error.html", http.StatusFound)
        p.errorLog(http.StatusFound, "empty input")
        return
    }
    
    serverID, serverIP, err := getServerIPbyNum(serverNum)
    
    if err != nil {
        http.Redirect(p.w, p.r, *cfg.baseuri+"panic.html", http.StatusFound)
        p.errorLog(http.StatusFound, "error resolving '%s' ('%s', '%s'): %s", serverNum, serverID, serverIP, err.Error())
        return
    }
    
    if serverID == "" {
        http.Redirect(p.w, p.r, "error.html", http.StatusFound)
        p.errorLog(http.StatusFound, "error resolving '%s' ('%s', '%s'): server not found", serverNum, serverID, serverIP)
        return
    }
    
    if serverIP == "" {
        http.Redirect(p.w, p.r, "retry.html?"+p.GetAuthToken("retry", serverID), http.StatusFound)
        p.errorLog(http.StatusFound, "error resolving '%s' ('%s', '%s'): server offline", serverNum, serverID, serverIP)
        return
    }
    
    http.Redirect(p.w, p.r, p.GetVncUrl(serverIP), http.StatusFound)
    p.errorLog(http.StatusFound, "resolved '%s' ('%s', '%s')", serverNum, serverID, serverIP)
}

func (p *Responder) GetVncUrl(serverIP string) string {
    wspath := strings.Trim(*cfg.baseuri, "/")
    wspath = wspath + ternary(len(wspath) > 0, "/", "").(string)
    return fmt.Sprintf(
        "%snoVNC/vnc_auto.html?title=%s&true_color=%s&cursor=%s&shared=%s&view_only=%s&path=%s",
        *cfg.baseuri,
        url.QueryEscape(serverIP),
        ternary(*cfg.vnc_true_color, "true", "false").(string),
        ternary(*cfg.vnc_local_cursor, "true", "false").(string),
        ternary(*cfg.vnc_shared, "true", "false").(string),
        ternary(*cfg.vnc_view_only, "true", "false").(string),
        url.QueryEscape(wspath+"websockify?"+p.GetAuthToken("dest", serverIP)),
    )
}


func (p *Responder) GetAuthToken(dest string, serverIP string) string {
    return base64.URLEncoding.EncodeToString(slidingPassword.Encrypt(fmt.Sprintf("peer %s real %s %s %s:%d", SplitHostPort(p.r.RemoteAddr)[0], p.getHeader("X-Real-IP", "UNKNOWN"), dest, serverIP, *cfg.vnc_port)))
}


func getServerID(serverNum string) (string, error) {
    client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
    
    resp, err := client.Get(*cfg.auth+url.QueryEscape(serverNum))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    buf, err := ioutil.ReadAll(io.LimitReader(resp.Body, 256))
    if err != nil && err != io.EOF {
        return "", err
    }

    return strings.Trim(string(buf), " \r\n"), nil
}

func getServerIP(serverID string) (string, error) {
    resp, err := http.Get(*cfg.resolv+url.QueryEscape(serverID))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    buf, err := ioutil.ReadAll(io.LimitReader(resp.Body, 256))
    if err != nil && err != io.EOF {
        return "", err
    }

    return strings.Trim(string(buf), " \r\n"), nil
}

func getServerIPbyNum(serverNum string) (string, string, error) {
    serverID, err := getServerID(serverNum)
    if serverID == "" {
        return "", "", err
    }
    
    serverIP, err := getServerIP(serverID)
    if serverIP == "" {
        return serverID, "", err
    }

    return serverID, serverIP, nil
}


func SplitHostPort(hostport string) [2]string {
    host, port, err := net.SplitHostPort(hostport)
    if err != nil {
    	panic(err)
    }
	return [2]string{host, port}
}

