/*
Copyright © 2022 Authors of Patu

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

#include <linux/bpf.h>

#include "include/helpers/helpers.h"
#include "include/helpers/maps.h"

static int subnetIP = 0;

static inline void extract_socket_key_v4(struct sk_msg_md *msg,
                                         struct socket_key *sockkey) {

  sockkey->src_ip = msg->remote_ip4;
  sockkey->dst_ip = msg->local_ip4;
  sockkey->src_port = msg->remote_port >> 16;
  sockkey->dst_port = (bpf_htonl(msg->local_port)) >> 16;
}

static inline int in_subnet_range(__u32 ip) {
  if (!subnetIP) {
    enum cni_config_key key = SUBNET_IP;
    union cni_config_value *ipAddr = map_lookup_elem(&cni_config_map, &key);
    if (ipAddr) {
      // Ip written in config map is in host order
      subnetIP = ipAddr->ipv4;
    }
  }
  if (subnetIP != 0) {
    return ((ip & bpf_htonl(0xffff0000)) == (subnetIP & bpf_htonl(0xffff0000)));
  }
  return 0;
}

__section("sk_msg") int patu_skmsg(struct sk_msg_md *msg) {
  if (in_subnet_range(msg->remote_ip4)) {

    struct socket_key sockkey = {};
    extract_socket_key_v4(msg, &sockkey);
    long result =
        msg_redirect_hash(msg, &sockops_redir_map, &sockkey, BPF_F_INGRESS);
    if (result) {
      print_info("[sk_msg], packets redirected from source port %d "
                 "to destination port %d",
                 (bpf_htonl(msg->local_port)) >> 16, msg->remote_port >> 16);
    }
  }
  return SK_PASS;
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
