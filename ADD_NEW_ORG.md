1. Follow the README.md
2. Create 2 new subdomains on route53 ( A or CNAME  register )
  - api.{newOrg}.sunchain.fr
  - explorer.{newOrg}.sunchain.fr
3. Copy CaddyFile in all VM (could automate with Ansible) (See below)
4. Run Caddy on each node (could automate with Ansible): 

```
docker run -d --restart always  -e ACME_AGREE=true  -v $(pwd)/Caddyfile:/etc/Caddyfile     -v $HOME/.caddy:/root/.caddy  --network="dockercompose_default"   -p 80:80 -p 443:443 abiosoft/caddy
```

3. Create credentials in app.sunchain.io
  - set blockchain field to 1 ( need to be migrated to a permission / role )
4. Add a line in blockchain settings in app.sunchain.io
5. Get the token with the credentials 

  

Cadyfile:

explorer.{newOrg}.sunchain.fr {
  tls julien.cappiello@sunchain.fr
#  basicauth / user pass
  proxy / explorer.{newOrg}.sunchain.io:8080 {
     transparent
  }
}

api.{newOrg}.sunchain.fr {
  tls julien.cappiello@sunchain.fr
  proxy / api.{newOrg}.sunchain.io:4000 {
    transparent
  }
}
