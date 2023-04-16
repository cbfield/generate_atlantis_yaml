module "module1" {
  source = "../modules/module1"
}

locals {
  thing = file("${path.module}/files/this.txt")
}
