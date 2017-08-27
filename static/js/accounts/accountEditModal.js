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
          return "/static/views/accounts/accountEditModal.html"
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
              Attributes: [],
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

          $scope.attrKind = function(kindID) {
            switch (kindID){
              case 0:
                return "GROUP";
            }
            return "?";
          }
          $scope.attrIcon = function(kindID) {
            switch (kindID){
              case 0:
                return "group";
            }
            return "?";
          }
          $scope.newAttr = function(kindID) {
            console.log($scope.account);
            if (!$scope.account.Attributes)$scope.account.Attributes = [];
            $scope.account.Attributes.push({
              Kind: kindID,
              Name: '',
              Val: '',
            });
          }
          $scope.deleteAttr = function(index){
            $scope.account.Attributes.splice(index, 1);
          }

          //Checks and applies validation classes. Returns true if fields are valid.
          function valid(){
            if (!$scope.account.Attributes)$scope.account.Attributes = [];
            var attrValidation = [];

            for(var i = 0 ; i < 12; i++) {
              $scope['attrVldn_' + i] = [];
            }

            for(var i = 0 ; i < $scope.account.Attributes.length; i++) {
              $scope['attrVldn_' + i] = validationClass($scope.account.Attributes[i].Name, false);
              attrValidation = attrValidation.concat($scope['attrVldn_' + i]);
            }
            $scope.nameVldn = $scope.usernameVldn = [];
            $scope.nameVldn = validationClass($scope.account.DisplayName, false);
            $scope.usernameVldn = validationClass($scope.account.Username, false);
            return !$scope.nameVldn.concat($scope.usernameVldn).concat(attrValidation).includes('invalid');
          };

        }
      };
  });
})();
