package panic

func Do( f func(), h func(r interface{}) ) {

     defer func() {
     
     	   if r := recover(); r != nil {
              h(r)
           }
     
     }()

     f()
     
}


