(function() {
  'use strict';

  var dataset = $('#dataset');
  var workers = $('#workers');
  var title = $('#title');
  var results = $('#results');
  var loading = $('#loading');
  var chart = $('#scatter');
  var footer = {
    creditCard: $('#credit-card-footer')
  };

  $('#submit').click(function() {
    var numWorkers = workers.val();
    if (!numWorkers) {
      return;
    }

    results.empty();
    chart.empty();
    loading.show();

    switch (dataset.val()) {
    case 'credit-card':
      title.text('Credit Card Defaults');
      $.get('/api/pca/credit-card/' + numWorkers, function(resp) {
        if (resp.status !== 'ok') {
          alert('Uh oh! ' + resp.message);
          return;
        }
        results.html('<p>Elapsed time: ' + resp.elapsed + ' seconds</p>');
        scatter('#scatter', '#loading', 'credit-card.csv');
        $('.footer').hide();
        footer.creditCard.show();
      });
      break;
    default:
      break;
    }
  });
})();
