(function () {

  function validationClass(value, shouldBeNum){
    if (value == undefined)return ['invalid'];
    if (value === '')return ['invalid'];
    if (shouldBeNum && isNaN(value))return ['invalid'];
    return ['valid'];
  }

    app.directive('accountEditModal', function($rootScope){
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
          return "/static/views/accountEditModal.html"
        },
        link: function($scope, elem, attrs) {
          // scope = either parent scope or its own child scope if scope set.
          // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
        },
        controller: function($scope) {
          $scope.open = false;
          $scope.isEditMode = false;
          $scope.account = {};
          $scope.cb = null;
          $scope.nameVldn = $scope.usernameVldn = [];

          $rootScope.$on('create-account', function(event, args) {
            $scope.open = true;
            $scope.isEditMode = false;
            $scope.account = {
              AdminPerms: {Accounts: false, Data: false, Integrations: false},
              IsRobot:false,
            };
            $scope.cb = args.cb;
          });

          $rootScope.$on('edit-account', function(event, args) {
            $scope.open = true;
            $scope.isEditMode = true;
            $scope.account = args.account;
            $scope.cb = args.cb;
          });

          $scope.setType = function(t){
            $scope.typeSelected = t;
          }

          $scope.onCancel = function(){
            $scope.open = false;
            $scope.isEditMode = false;
            $scope.cb = null;
            $scope.nameVldn = $scope.usernameVldn = '';
          }

          $scope.onFinish = function(){
            if (valid()){
              $scope.cb($scope.account);
              $scope.onCancel();
            }
          }

          //Checks and applies validation classes. Returns true if fields are valid.
          function valid(){
            $scope.nameVldn = $scope.usernameVldn = [];
            $scope.nameVldn = validationClass($scope.account.DisplayName, false);
            $scope.usernameVldn = validationClass($scope.account.Username, false);
            return !$scope.nameVldn.concat($scope.usernameVldn).includes('invalid');
          };

        }
      };
  });
})();
