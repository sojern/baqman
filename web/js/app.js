$('#killall').click(function(event) {
    if(this.checked) {
        $("input[name='jobkill']").each(function() {
            this.checked = true;
        });
    } else {
        $("input[name='jobkill']").each(function() {
            this.checked = false;
        });
    }
});
