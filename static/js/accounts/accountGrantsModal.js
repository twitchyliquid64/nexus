(function () {

    app.directive('accountGrantsModal', function($rootScope){
      return {
        //scope allows us to setup variable bindings to the parent scope. By default, we share the parent scope. For an isolated one, we should
        //pass an object as the scope attribute which has a dict of the variable name for us, and a string describing where and how to bind it.
        //scope: {color: '@colorAttr'} binds 'color' to the value of color-attr with a one way binding. Only strings supported here: color-attr="{{color}}"
        //scope: {color: '=color'} binds 'color' two way. Pass the object here: color="color".
        scope: {

        },
        //restrict E means its can only be used as an element.
        restrict: 'E',
        templateUrl: function(elem, attr){
          return "/static/views/accounts/accountGrantsModal.html"
        },
        link: function($scope, elem, attrs) {
          // scope = either parent scope or its own child scope if scope set.
          // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
        },
        controller: function($scope) {
          $scope.open = false;
          $scope.account = {};
          $scope.cb = null;
          $scope.rw = false;
          $scope.dsuid = '';

          $rootScope.$on('open-user-grants', function(event, args) {
            $scope.open = true;
            $scope.account = args.account;
            $scope.cb = args.cb;
            $scope.rw = false;
            $scope.dsuid = '';
          });

          $scope.readOnlyStr = function(grant){
            if (grant.ReadOnly)return "read-only";
            return "r/w";
          }

          $scope.doAdd = function(){
            if ($scope.dsuid) {
              $scope.cb({'action': 'add', 'dsuid': parseInt($scope.dsuid), 'rw': $scope.rw, 'uid': $scope.account.UID});
              $scope.onCancel();
            }
          }

          $scope.doDelete = function(grant){
            $scope.cb({'action': 'delete', 'gid': grant.UID});
            $scope.onCancel();
          }

          $scope.onCancel = function(){
            $scope.open = false;
          }
        }
      };
  });
})();
